// CRC: crc-LuaSession.md, crc-Server.md (each Session owns its LuaSession)
// Spec: deployment.md, interfaces.md
// Sequence: seq-server-startup.md, seq-session-create-backend.md, seq-lua-session-init.md
//
// Server implements per-session Lua isolation: each Session holds its own LuaSession.
// It creates/destroys LuaSessions via SessionManager callbacks and implements
// PathVariableHandler to route HandleFrontendCreate/Update to per-session LuaSession.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	gopher "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/backend"
	"github.com/zot/ui-engine/internal/bundle"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/lua"
	"github.com/zot/ui-engine/internal/protocol"
	"github.com/zot/ui-engine/internal/site"
	"github.com/zot/ui-engine/internal/viewdef"
)

// Server is the main UI server.
// CRC: crc-Server.md
type Server struct {
	config           *config.Config
	sessions         *SessionManager
	handler          *protocol.Handler
	pendingQueues    *PendingQueueManager
	httpServer       *http.Server
	HttpEndpoint     *HTTPEndpoint
	wsEndpoint       *WebSocketEndpoint
	backendSocket    *BackendSocket
	luaConfig        *luaSetupConfig // Shared config for creating new sessions
	wrapperRegistry  *lua.WrapperRegistry
	storeAdapter     *luaTrackerAdapter
	viewdefManager   *viewdef.ViewdefManager
	hotLoader        *lua.HotLoader     // Lua hot-reloading (nil if disabled)
	viewdefHotLoader *viewdef.HotLoader // Viewdef hot-reloading (nil if disabled)
	dangerousMu      sync.Mutex
	dangerousSites   map[string]bool // call sites already warned about (R170)
}

// luaSetupConfig holds shared configuration for creating Lua sessions.
type luaSetupConfig struct {
	config      *config.Config
	luaDir      string
	mainLuaCode string // Cached main.lua for bundle mode
}

// New creates a new server with the given configuration.
func New(cfg *config.Config) *Server {
	sessions := NewSessionManager(cfg.Session.Timeout.Duration())
	s := &Server{
		config:        cfg,
		sessions:      sessions,
		pendingQueues: NewPendingQueueManager(),
	}
	// Create message sender that wraps WebSocket endpoint
	sender := &serverMessageSender{server: s}
	s.handler = protocol.NewHandler(cfg, sender)

	// Set up pending queue for CLI/REST clients
	s.handler.SetPendingQueuer(s.pendingQueues)

	// Set up backend lookup for per-session watch management
	s.handler.SetBackendLookup(&serverBackendLookup{server: s})

	// Route handler outgoing messages through session batchers
	s.handler.SetQueuer(&serverMessageQueuer{server: s})

	// Create WebSocket endpoint
	s.wsEndpoint = NewWebSocketEndpoint(cfg, sessions, s.handler)

	// Create HTTP endpoint
	s.HttpEndpoint = NewHTTPEndpoint(sessions, s.handler, s.wsEndpoint)

	// Set up site serving (bundle or custom directory)
	s.setupSite(cfg)

	// Set up viewdef manager and load viewdefs
	s.setupViewdefs(cfg)

	// Create backend socket
	s.backendSocket = NewBackendSocket(cfg, cfg.Server.Socket, s.handler, s.HttpEndpoint)

	// Set verbosity on all components
	// Note: Components now use Config.Log directly via the passed config object.
	// verbosity := cfg.Verbosity() - Removed
	// s.wsEndpoint.SetVerbosity(verbosity) - Removed
	// s.backendSocket.SetVerbosity(verbosity) - Removed
	// s.handler.SetVerbosity(verbosity) - Removed
	// store.SetVerbosity(verbosity) - Removed

	// Initialize wrapper registry (needed for ViewList wrapper support)
	s.wrapperRegistry = lua.NewWrapperRegistry()

	// Initialize Lua runtime if enabled
	if cfg.Lua.Enabled {
		s.setupLua(cfg)

		// Set up debug data provider for /debug/variables page
		s.HttpEndpoint.SetDebugDataProvider(func(sessionID string, diagLevel int) ([]DebugVariable, int64, error) {
			luaSession, _ := s.luaSessionFor(sessionID)
			if luaSession == nil {
				return nil, 0, fmt.Errorf("session %s not found", sessionID)
			}
			tracker := luaSession.GetTracker()
			if tracker == nil {
				return nil, 0, fmt.Errorf("tracker not found")
			}
			vars, err := s.getDebugVariables(tracker)
			return vars, tracker.ChangeCount, err
		})

		// Set viewdef manager on store adapter so it can send viewdefs when new types appear
		if s.storeAdapter != nil && s.viewdefManager != nil {
			s.storeAdapter.SetViewdefManager(s.viewdefManager)
		}

		// Note: Go wrappers (like ViewList) auto-register via init() - no explicit registration needed

		// Set session callbacks for Lua session management
		// Callbacks receive vended IDs (compact integers) for backend communication
		// Each session gets its own LuaBackend and OutgoingBatcher for per-session isolation
		sessions.SetOnSessionCreated(func(vendedID string, sess *Session) error {
			return s.CreateLuaBackendForSession(vendedID, sess)
		})
		sessions.SetOnSessionDestroyed(func(vendedID string, sess *Session) {
			s.DestroyLuaBackendForSession(vendedID, sess)
		})

		// Set up afterBatch callback for automatic change detection
		s.wsEndpoint.SetAfterBatch(s.AfterBatch)

		// Set up disconnect callback to clear sent-tracking and stale variables for page refresh
		s.wsEndpoint.SetOnDisconnect(func(internalSessionID string) {
			// Translate internal UUID to vended ID for viewdef manager
			vendedID := s.sessions.GetVendedID(internalSessionID)
			if vendedID != "" && s.viewdefManager != nil {
				s.viewdefManager.ClearSession(vendedID)
			}

			// Clear all descendants of the app variable so page refresh starts fresh
			if vendedID != "" && s.storeAdapter != nil {
				if lb := s.storeAdapter.GetBackend(vendedID); lb != nil {
					if luaSession := s.GetLuaSession(vendedID); luaSession != nil {
						rootID := luaSession.GetAppVariableID()
						if rootID != 0 {
							lb.ClearDescendants(rootID)
						}
					}
				}
			}
		})

		// Set server as path variable handler (routes to per-session LuaSession)
		s.handler.SetPathVariableHandler(s)
	}

	return s
}

// Start starts the server.
func (s *Server) Start() error {
	// Start HTTP server and block
	srv, ln, url, err := s.configureHTTP(s.config.Server.Port)
	if err != nil {
		return err
	}
	s.config.Log(0, "HTTP server listening on %s", url)
	s.config.Log(0, "Serving site from directory: %s", s.HttpEndpoint.staticDir)
	// Block until shutdown
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %v", err)
	}
	return nil
}

// configureHTTP sets up the backend socket and HTTP server listener.
// Returns the server, listener, and base URL. A configured site directory
// that does not exist is an error, returned before anything listens.
// CRC: crc-Server.md | Seq: seq-app-discovery.md#1.1 | R119
func (s *Server) configureHTTP(port int) (*http.Server, net.Listener, string, error) {
	if dir := s.config.Server.Dir; dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return nil, nil, "", fmt.Errorf("site directory %s does not exist", dir)
		}
	}
	// Start backend socket
	if err := s.backendSocket.Listen(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to start backend socket: %w", err)
	}
	s.config.Log(0, "Backend socket listening on %s", s.backendSocket.GetSocketPath())

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.HttpEndpoint,
	}

	// We need to capture the actual port if 0 was passed
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Update port in config if it was 0
	if port == 0 {
		addr = listener.Addr().String()
		_, portStr, _ := net.SplitHostPort(addr)
		s.config.Server.Port, _ = strconv.Atoi(portStr)
	}

	host := s.config.Server.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}

	url := fmt.Sprintf("http://%s:%d", host, s.config.Server.Port)
	return s.httpServer, listener, url, nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop hot loader first
	if s.hotLoader != nil {
		s.hotLoader.Stop()
		s.hotLoader = nil
	}

	// Shut down all Lua sessions. The registry snapshot releases its lock before any
	// Shutdown(): Shutdown joins its executor, which may run an in-flight deferred task
	// that re-enters the Server (e.g. the app-loader resolving a session). Holding the
	// registry lock across the join would deadlock such a task. Afterwards every Lua
	// session reports shut down, so luaSessionFor finds nothing.
	// CRC: crc-LuaSession.md, crc-Server.md | Seq: seq-session-timer.md | R118, R233
	for vendedID, luaSession := range s.runningLuaSessions() {
		s.config.Log(0, "Shutting down Lua session %s", vendedID)
		luaSession.Shutdown()
	}

	// Close backend socket
	if s.backendSocket != nil {
		s.backendSocket.Close()
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// GetSessions returns the session manager.
func (s *Server) GetSessions() *SessionManager {
	return s.sessions
}

// GetHandler returns the protocol handler.
func (s *Server) GetHandler() *protocol.Handler {
	return s.handler
}

// StartCleanupWorker starts a background worker to clean up inactive sessions.
func (s *Server) StartCleanupWorker(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			count := s.sessions.CleanupInactiveSessions()
			if count > 0 {
				s.config.Log(0, "Cleaned up %d inactive sessions", count)
			}
		}
	}()
}

// setupSite configures the site filesystem (bundle or directory).
func (s *Server) setupSite(cfg *config.Config) {
	// If --dir is specified, use that directory's html/ subdirectory
	if cfg.Server.Dir != "" {
		htmlDir := cfg.Server.Dir + "/html"
		s.HttpEndpoint.SetStaticDir(htmlDir)
		// CRC: crc-Server.md | R138, R140
		s.HttpEndpoint.SetAppsDir(filepath.Join(cfg.Server.Dir, "lua"))
		s.config.Log(0, "Serving site from directory: %s", htmlDir)
		return
	}

	// Try to load from bundle
	zipReader, err := bundle.GetBundleReader()
	if err != nil {
		s.config.Log(0, "Warning: failed to read bundle: %v", err)
		return
	}

	if zipReader != nil {
		// NewZipFileSystem automatically serves from html/ subdirectory
		s.HttpEndpoint.SetEmbeddedSite(bundle.NewZipFileSystem(zipReader))
		s.config.Log(0, "Serving site from embedded bundle (html/)")
		return
	}

	s.config.Log(0, "Warning: no site available (not bundled and no --dir specified)")
}

// setupViewdefs initializes the viewdef manager and loads viewdefs: from
// the active apps and the site's own viewdefs/ when a site directory is
// configured, else from the bundle. A missing viewdefs/ is not logged.
// CRC: crc-Server.md | Seq: seq-app-discovery.md#1.2, seq-app-discovery.md#1.3, seq-app-discovery.md#1.4 | R120, R125
func (s *Server) setupViewdefs(cfg *config.Config) {
	s.viewdefManager = viewdef.NewViewdefManager()
	s.viewdefManager.SetLogger(cfg.Log)

	if siteDir := cfg.Server.Dir; siteDir != "" {
		s.viewdefManager.SetViewdefDir(filepath.Join(siteDir, "viewdefs"))
		apps := site.Discover(siteDir, cfg.Server.BuiltinApps)
		s.viewdefManager.LoadFromSources(site.ViewdefSources(siteDir, apps))
		s.config.Log(0, "Loaded %d viewdefs from %d apps in %s", s.viewdefManager.Count(), len(apps), siteDir)

		// Initialize viewdef hot-loader if Lua hot-loading is enabled
		// (reuse the same config flag for viewdefs)
		if cfg.Lua.Hotload {
			s.setupViewdefHotLoader(cfg)
		}
		return
	}

	// Try to load from bundle
	if err := s.viewdefManager.LoadFromBundle(); err != nil {
		s.config.Log(0, "Warning: failed to load viewdefs from bundle: %v", err)
	} else if s.viewdefManager.Count() > 0 {
		s.config.Log(0, "Loaded %d viewdefs from bundle", s.viewdefManager.Count())
	}
}

// setupViewdefHotLoader initializes the viewdef hot-loader.
// CRC: crc-ViewdefStore.md
// Sequence: seq-viewdef-hotload.md
func (s *Server) setupViewdefHotLoader(cfg *config.Config) {
	hotLoader, err := viewdef.NewHotLoader(
		cfg,
		s.viewdefManager,
		s, // Server implements viewdef.SessionPusher
	)
	if err != nil {
		s.config.Log(0, "ViewdefHotLoader: failed to create: %v", err)
		return
	}

	s.viewdefHotLoader = hotLoader
	if err := hotLoader.Start(); err != nil {
		s.config.Log(0, "ViewdefHotLoader: failed to start: %v", err)
		s.viewdefHotLoader = nil
		return
	}

	s.config.Log(0, "ViewdefHotLoader: watching viewdefs of %s for changes", cfg.Server.Dir)
}

// GetSessionIDs returns all active vended session IDs.
// Implements viewdef.SessionPusher.
// CRC: crc-Server.md | R233
func (s *Server) GetSessionIDs() []string {
	sessions := s.runningLuaSessions()
	ids := make([]string, 0, len(sessions))
	for vendedID := range sessions {
		ids = append(ids, vendedID)
	}
	return ids
}

// PushViewdefs pushes updated viewdefs to a session.
// This triggers AfterBatch to detect and send the changes.
// Implements viewdef.SessionPusher.
// CRC: crc-ViewdefStore.md
// Sequence: seq-viewdef-hotload.md
func (s *Server) PushViewdefs(vendedID string, viewdefs map[string]string) {
	// Execute in session to trigger AfterBatch
	// The viewdef manager already has the updated content, so AfterBatch will pick it up
	s.ExecuteInSession(vendedID, func() (interface{}, error) {
		// No-op - just trigger AfterBatch which will detect the changed viewdef
		return nil, nil
	})
}

// triggerSessionRefresh triggers AfterBatch for a session to push pending changes.
// Used by Lua hot-loader after reloading code.
// CRC: crc-LuaHotLoader.md
// Sequence: seq-lua-hotload.md
func (s *Server) triggerSessionRefresh(vendedID string) {
	s.config.Log(1, "triggerSessionRefresh: triggering refresh for session %s", vendedID)
	s.ExecuteInSession(vendedID, func() (interface{}, error) {
		// No-op - just trigger AfterBatch which will detect any changes
		return nil, nil
	})
}

// SetSiteFS sets a custom filesystem for serving static files.
func (s *Server) SetSiteFS(siteFS fs.FS) {
	s.HttpEndpoint.SetEmbeddedSite(siteFS)
}

// SetRootSessionProvider sets a provider for the root path "/" session.
// If the provider returns a session ID, that session is used instead of creating a new one.
// This allows MCP-style servers to serve an existing session at "/" without redirect.
func (s *Server) SetRootSessionProvider(provider RootSessionProvider) {
	s.HttpEndpoint.SetRootSessionProvider(provider)
}

// serverMessageSender implements protocol.MessageSender.
type serverMessageSender struct {
	server *Server
}

func (sms *serverMessageSender) Send(connectionID string, msg *protocol.Message) error {
	return sms.server.wsEndpoint.Send(connectionID, msg)
}

func (sms *serverMessageSender) Broadcast(sessionID string, msg *protocol.Message) error {
	return sms.server.wsEndpoint.Broadcast(sessionID, msg)
}

// serverMessageQueuer implements protocol.MessageQueuer.
// It routes messages through the session's OutgoingBatcher for a given connection.
// CRC: crc-ProtocolHandler.md | R112, R113
type serverMessageQueuer struct {
	server *Server
}

func (smq *serverMessageQueuer) Queue(msg *protocol.Message, watchers []string) {
	// All watchers share the same session, so use the first to look up the batcher.
	if len(watchers) == 0 {
		smq.server.config.Log(0, "QUEUER: no watchers")
		return
	}
	sessionID := smq.server.wsEndpoint.GetSessionIDForConnection(watchers[0])
	if sessionID == "" {
		smq.server.config.Log(0, "QUEUER: no session for connection %s", watchers[0])
		return
	}
	sess := smq.server.sessions.Get(sessionID)
	if sess == nil {
		smq.server.config.Log(0, "QUEUER: session %s not found", sessionID)
		return
	}
	if batcher := sess.GetBatcher(); batcher != nil {
		smq.server.config.Log(0, "QUEUER: routing %s through batcher for session %s", msg.Type, sessionID)
		// Enqueue without starting a timer — AfterBatch handles flush
		batcher.Enqueue(msg, watchers)
	} else {
		smq.server.config.Log(0, "QUEUER: no batcher, sending directly for session %s", sessionID)
		for _, connID := range watchers {
			smq.server.wsEndpoint.Send(connID, msg)
		}
	}
}

// serverBackendLookup implements protocol.BackendLookup.
// It looks up the session's backend for a given connection ID.
type serverBackendLookup struct {
	server *Server
}

func (sbl *serverBackendLookup) GetBackendForConnection(connectionID string) backend.Backend {
	// Look up session by connection ID via WebSocket endpoint
	sessionID := sbl.server.wsEndpoint.GetSessionIDForConnection(connectionID)
	if sessionID == "" {
		return nil
	}

	// Get session
	sess := sbl.server.sessions.Get(sessionID)
	if sess == nil {
		return nil
	}

	return sess.GetBackend()
}

// setupLua initializes the Lua runtime.
// Only main.lua is auto-loaded per session. Other Lua files are loaded
// on-demand via require() or the lua property on variable 1.
func (s *Server) setupLua(cfg *config.Config) {
	// Determine Lua directory
	luaDir := cfg.Lua.Path
	if cfg.Server.Dir != "" {
		luaDir = filepath.Join(cfg.Server.Dir, "lua")
	}

	// Initialize shared config
	s.luaConfig = &luaSetupConfig{
		config: cfg,
		luaDir: luaDir,
	}

	// Create store adapter (will be shared across sessions)
	s.storeAdapter = &luaTrackerAdapter{config: cfg, sessions: s.sessions}

	// In bundle mode, pre-cache main.lua content for per-session execution
	if cfg.Server.Dir == "" {
		s.preloadMainLuaFromBundleToConfig()
	}

	// Initialize hot loader if enabled
	if cfg.Lua.Hotload {
		hotLoader, err := lua.NewHotLoader(cfg, luaDir, s.getLuaSessions, s.triggerSessionRefresh)
		if err != nil {
			s.config.Log(0, "HotLoader: failed to create: %v", err)
		} else {
			s.hotLoader = hotLoader
			if err := hotLoader.Start(); err != nil {
				s.config.Log(0, "HotLoader: failed to start: %v", err)
				s.hotLoader = nil
			}
		}
	}

	s.config.Log(0, "Lua sessions enabled (dir: %s, hotload: %v)", luaDir, cfg.Lua.Hotload)
}

// CreateLuaBackendForSession creates a LuaBackend and LuaSession for a new frontend session.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
// Each frontend session gets its own isolated Lua state and OutgoingBatcher.
// CRC: crc-LuaBackend.md
// Sequence: seq-session-create-backend.md
func (s *Server) CreateLuaBackendForSession(vendedID string, sess *Session) error {
	if s.luaConfig == nil {
		return nil // Lua not enabled
	}

	// Create a new LuaSession with its own Lua state
	luaSession, err := lua.NewRuntime(s.luaConfig.config, s.luaConfig.luaDir, s.viewdefManager)
	if err != nil {
		return fmt.Errorf("failed to create Lua session: %w", err)
	}

	// Set cached main.lua code if available (bundle mode)
	if s.luaConfig.mainLuaCode != "" {
		luaSession.SetMainLuaCode(s.luaConfig.mainLuaCode)
	}

	// Set wrapper registry on session (allows ui.registerWrapper from Lua)
	luaSession.SetWrapperRegistry(s.wrapperRegistry)

	// Set defer callback for session timers (setImmediate/setTimeout/setInterval)
	// Seq: seq-session-timer.md
	luaSession.SetDeferCallback(func(fn func() (interface{}, error)) {
		s.ExecuteInSessionAsync(vendedID, fn)
	})

	// Create LuaBackend with resolver
	lb := backend.NewLuaBackend(s.config, vendedID, &lua.LuaResolver{})

	// Attach backend to session
	sess.SetBackend(lb)

	// Create per-session outgoing batcher
	// Each session has its own batcher for isolated debouncing
	sess.SetBatcher(NewOutgoingBatcher(s.wsEndpoint))

	// Set variable store on Lua session
	luaSession.SetVariableStore(s.storeAdapter)

	// Attach to the session before main.lua runs, so the session can be found while it
	// starts up; the session is its only holder.
	// CRC: crc-Server.md, crc-Session.md | Seq: seq-session-create-backend.md | R230, R231
	sess.attachLua(luaSession)

	// Initialize the session (creates session table, runs main.lua)
	_, err = luaSession.CreateLuaSession(vendedID)
	if err != nil {
		sess.SetBackend(nil)
		return err
	}

	s.config.Log(0, "Created Lua session %s with isolated state", vendedID)
	return nil
}

// DestroyLuaBackendForSession destroys a session's LuaBackend and LuaSession.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
func (s *Server) DestroyLuaBackendForSession(vendedID string, sess *Session) {
	if s.luaConfig == nil {
		return
	}

	// Close the session's open Tailors first: cancel their contexts and log them.
	// CRC: crc-Server.md | Seq: seq-tailor.md#2.2.1 | R193
	for _, t := range sess.openTailors() {
		t.close("session destroyed", true)
	}

	// The session holds its Lua session; there is no map entry to remove.
	// CRC: crc-Server.md | R231
	luaSession := sess.LuaSession()

	// Shutdown the Lua session (closes Lua state)
	if luaSession != nil {
		luaSession.Shutdown()
	}

	// Shutdown backend
	if b := sess.GetBackend(); b != nil {
		b.Shutdown()
	}
	sess.SetBackend(nil)

	s.config.Log(0, "Destroyed Lua session %s", vendedID)
}

// AfterBatch triggers Lua change detection after processing a message batch.
// internalSessionID is the full UUID session ID (used in URLs/WebSocket bindings).
// userEvent indicates if the batch was triggered by user interaction (immediate flush needed).
// This method looks up the vended ID and calls the Lua session's AfterBatch,
// then queues detected changes to the outgoing batcher.
// CRC: crc-Server.md
// Sequence: seq-relay-message.md, seq-backend-refresh.md, seq-frontend-outgoing-batch.md
func (s *Server) AfterBatch(internalSessionID string, userEvent bool) {
	// Get session, its backend, and batcher
	sess := s.sessions.Get(internalSessionID)
	if sess == nil {
		return
	}

	b := sess.GetBackend()
	if b == nil {
		return
	}

	batcher := sess.GetBatcher()

	vendedID := s.sessions.GetVendedID(internalSessionID)
	if vendedID == "" {
		return
	}

	// AfterBatch has the session already; ask it for its Lua session.
	// CRC: crc-Server.md | R232
	luaSession := sess.LuaSession()
	if luaSession == nil || luaSession.IsShutdown() {
		return
	}

	// Get detected changes from Lua session
	updates := luaSession.AfterBatch(vendedID)
	if len(updates) == 0 {
		// Even with no updates, flush immediately for user events
		if userEvent && batcher != nil {
			batcher.FlushNow()
		}
		return
	}

	// Queue each update to batcher or send directly
	for _, update := range updates {
		watchers := b.GetWatchers(update.VarID)
		if len(watchers) == 0 {
			continue
		}

		// Build update message
		updateMsg, err := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
			VarID:      update.VarID,
			Value:      update.Value,
			Properties: update.Properties,
		})
		if update.Properties["viewdefs"] != "" {
			s.config.Log(4, "SENDING VIEWDEFS TO ENDPOINT: %#v", update.Properties)
		}
		if err != nil {
			continue
		}

		// Queue to batcher or send directly
		if batcher != nil {
			batcher.Queue(updateMsg, watchers)
		} else {
			// Fallback: send directly (no batching)
			for _, connID := range watchers {
				s.wsEndpoint.Send(connID, updateMsg)
			}
		}
	}

	// Flush immediately for user events
	if userEvent && batcher != nil {
		batcher.FlushNow()
	}
}

// ExecuteInSession executes code within a session's context and waits for it.
// It is dangerous: called from inside an executor it deadlocks, and nothing in
// the call shows which kind of caller it is. So it delegates to
// PrimitiveExecuteInSession with dangerous set, which warns once per calling
// site or panics, per Config.Session.AllowDangerousExecution.
// vendedID is the compact session ID ("1", "2", etc.)
// CRC: crc-Server.md | Seq: seq-session-ref.md#2.1.1 | R168
func (s *Server) ExecuteInSession(vendedID string, fn func() (interface{}, error)) (interface{}, error) {
	return s.PrimitiveExecuteInSession(vendedID, fn, true)
}

// executeInSession is ExecuteInSession's behaviour from before dangerous
// execution existed: queue through the session's executor, set the Lua
// session context, wait, and run afterBatch when a browser is connected.
// CRC: crc-Server.md | Seq: seq-session-ref.md#2.2.4 | R169
func (s *Server) executeInSession(vendedID string, fn func() (interface{}, error)) (interface{}, error) {
	// Find the session the same way a reference does (R236).
	luaSession, sess := s.luaSessionFor(vendedID)
	if luaSession == nil {
		return nil, fmt.Errorf("session %s not found", vendedID)
	}
	internalID := sess.ID

	// Delegate to websocket endpoint (queues through session's executor)
	// Wrap fn to set up Lua session context
	return s.wsEndpoint.ExecuteInSession(internalID, func() (interface{}, error) {
		return luaSession.ExecuteInSession(vendedID, fn)
	})
}

// ExecuteInSessionAsync is a fire-and-forget variant of ExecuteInSession.
// It queues execution through ChanSvc without blocking the caller.
// Used by session timers (setImmediate/setTimeout/setInterval) to avoid deadlock
// when Lua code schedules deferred execution from within the executor.
// It is ExecRig.PostUpdate on a reference to the session, and does nothing when
// there is no such session.
// CRC: crc-Server.md, crc-LuaSession.md | Seq: seq-session-timer.md | R235
func (s *Server) ExecuteInSessionAsync(vendedID string, fn func() (interface{}, error)) {
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		return
	}
	ref.Exec().PostUpdate(func() { fn() })
}

// GetLuaSession returns a Lua session by vended ID (for testing/advanced use).
// CRC: crc-Server.md | R232
func (s *Server) GetLuaSession(vendedID string) *lua.LuaSession {
	ls, _ := s.luaSessionFor(vendedID)
	return ls
}

// luaSessionFor is the one way the server finds a Lua session: from the vended ID to
// the session, then the session's attached Lua session. It returns nil when there is no
// such session, no Lua session is attached yet, or it has shut down, which is what
// makes every lookup fail after Server.Shutdown. The server keeps no table of its own.
// CRC: crc-Server.md | Seq: seq-session-ref.md#1.1.1 | R229, R231, R232
func (s *Server) luaSessionFor(vendedID string) (*lua.LuaSession, *Session) {
	sess := s.sessions.GetByVendedID(vendedID)
	if sess == nil {
		return nil, nil
	}
	ls := sess.LuaSession()
	if ls == nil || ls.IsShutdown() {
		return nil, nil
	}
	return ls, sess
}

// runningLuaSessions returns every registered session's Lua session that is attached
// and running, keyed by vended ID, read from the session registry.
// CRC: crc-Server.md | R233
func (s *Server) runningLuaSessions() map[string]*lua.LuaSession {
	sessions := make(map[string]*lua.LuaSession)
	for _, e := range s.sessions.Snapshot() {
		if ls := e.Session.LuaSession(); ls != nil && !ls.IsShutdown() {
			sessions[e.VendedID] = ls
		}
	}
	return sessions
}

// getLuaSessions returns all active Lua sessions (used by HotLoader).
// CRC: crc-Server.md | R233
func (s *Server) getLuaSessions() []*lua.LuaSession {
	running := s.runningLuaSessions()
	sessions := make([]*lua.LuaSession, 0, len(running))
	for _, ls := range running {
		sessions = append(sessions, ls)
	}
	return sessions
}

// HandleFrontendCreate implements PathVariableHandler.
// It delegates to the per-session LuaSession.
// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
func (s *Server) HandleFrontendCreate(sessionID string, id int64, parentID int64, properties map[string]string) error {
	luaSession, _ := s.luaSessionFor(sessionID)
	if luaSession == nil {
		return fmt.Errorf("Lua session %s not found", sessionID)
	}
	return luaSession.HandleFrontendCreate(sessionID, id, parentID, properties)
}

// HandleFrontendUpdate implements PathVariableHandler.
// It delegates to the per-session LuaSession.
func (s *Server) HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage, properties map[string]string) error {
	luaSession, _ := s.luaSessionFor(sessionID)
	if luaSession == nil {
		return fmt.Errorf("Lua session %s not found", sessionID)
	}
	return luaSession.HandleFrontendUpdate(sessionID, varID, value, properties)
}

// getDebugVariables returns all variables in topological order from a tracker.
// CRC: crc-HTTPEndpoint.md (R57, R59, R60, R61)
func (s *Server) getDebugVariables(tracker *changetracker.Tracker) ([]DebugVariable, error) {
	allVars := tracker.Variables()

	// Build map for quick lookup and depth computation
	varMap := make(map[int64]*changetracker.Variable)
	for _, v := range allVars {
		varMap[v.ID] = v
	}

	// Compute depth for each variable
	depthMap := make(map[int64]int)
	var computeDepth func(id int64) int
	computeDepth = func(id int64) int {
		if d, ok := depthMap[id]; ok {
			return d
		}
		v := varMap[id]
		if v == nil || v.ParentID == 0 {
			depthMap[id] = 0
			return 0
		}
		d := computeDepth(v.ParentID) + 1
		depthMap[id] = d
		return d
	}
	for id := range varMap {
		computeDepth(id)
	}

	// Topological sort: BFS from roots
	var sorted []*changetracker.Variable
	visited := make(map[int64]bool)
	queue := make([]*changetracker.Variable, 0)
	for _, v := range allVars {
		if v.ParentID == 0 {
			queue = append(queue, v)
		}
	}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		if visited[v.ID] {
			continue
		}
		visited[v.ID] = true
		sorted = append(sorted, v)
		for _, childID := range v.ChildIDs {
			if child := varMap[childID]; child != nil && !visited[childID] {
				queue = append(queue, child)
			}
		}
	}
	// Add any orphans
	for _, v := range allVars {
		if !visited[v.ID] {
			sorted = append(sorted, v)
		}
	}

	// Convert to DebugVariable
	result := make([]DebugVariable, len(sorted))
	for i, v := range sorted {
		displayValue := v.WrapperJSON
		if displayValue == nil {
			displayValue = v.ValueJSON
		}
		goType := ""
		if v.Value != nil {
			goType = tracker.Resolver.GetType(v, v.Value)
		}
		info := DebugVariable{
			ID:          v.ID,
			ParentID:    v.ParentID,
			Value:       displayValue,
			BaseValue:   v.ValueJSON,
			Type:        v.Properties["type"],
			GoType:      goType,
			Path:        v.Properties["path"],
			Properties:  v.Properties,
			ChildIDs:    v.ChildIDs,
			Active:      v.Active,
			Access:      v.Properties["access"],
			ChangeCount: v.ChangeCount,
			Depth:       depthMap[v.ID],
			ElementId:   v.Properties["elementId"],
		}
		if v.ComputeTime > 0 {
			info.ComputeTime = formatDuration(v.ComputeTime)
		}
		if v.MaxComputeTime > 0 {
			info.MaxComputeTime = formatDuration(v.MaxComputeTime)
		}
		if len(v.Diags) > 0 {
			info.Diags = v.Diags
		}
		if v.Error != nil {
			info.Error = v.Error.Error()
		}
		result[i] = info
	}

	return result, nil
}

// formatDuration formats a time.Duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.3fs", d.Seconds())
}

// GetViewdefManager returns the viewdef manager.
func (s *Server) GetViewdefManager() *viewdef.ViewdefManager {
	return s.viewdefManager
}

// ViewdefCollisions returns the running server's live viewdef collisions.
// CRC: crc-Server.md | Seq: seq-app-discovery.md#3.1 | R143
func (s *Server) ViewdefCollisions() []viewdef.Collision {
	return s.viewdefManager.Collisions()
}

// StartAsync starts the HTTP server in a goroutine and returns the URL.
// Use this for MCP mode where the server runs alongside stdio MCP.
func (s *Server) StartAsync(port int) (string, error) {
	srv, ln, url, err := s.configureHTTP(port)
	if err != nil {
		return "", err
	}

	s.config.Log(0, "HTTP server listening on %s", url)
	s.config.Log(0, "Serving site from directory: %s", s.HttpEndpoint.staticDir)

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.config.Log(0, "HTTP server error: %v", err)
		}
	}()

	return url, nil
}

// preloadMainLuaFromBundleToConfig caches main.lua from bundle if available.
// The cached code is stored in luaConfig for per-session use.
func (s *Server) preloadMainLuaFromBundleToConfig() {
	content, err := bundle.ReadFile("lua/main.lua")
	if err != nil {
		// No main.lua in bundle - OK for hybrid/backend-only modes
		return
	}
	s.luaConfig.mainLuaCode = string(content)
	s.config.Log(0, "Preloaded main.lua from bundle")
}

// luaTrackerAdapter is the server's lua.VariableStore. It holds no per-session state:
// every operation names its session, and the adapter finds that session's backend
// through the session registry. There is no variable-to-session index, because variable
// IDs name a variable only within its session.
// CRC: crc-LuaTrackerAdapter.md | R239, R241
type luaTrackerAdapter struct {
	config         *config.Config
	viewdefManager *viewdef.ViewdefManager
	sessions       *SessionManager // the registry every lookup goes through (R239)
}

// SetViewdefManager sets the viewdef manager for sending viewdefs to frontend.
func (a *luaTrackerAdapter) SetViewdefManager(vm *viewdef.ViewdefManager) {
	a.viewdefManager = vm
}

// backend returns the Lua backend of the session with this vended ID, or nil when there
// is no such session or it has no Lua backend. The one way the adapter reaches a
// session's backend; it keeps no copy.
// CRC: crc-LuaTrackerAdapter.md | R239
func (a *luaTrackerAdapter) backend(vendedID string) *backend.LuaBackend {
	sess := a.sessions.GetByVendedID(vendedID)
	if sess == nil {
		return nil
	}
	lb, _ := sess.GetBackend().(*backend.LuaBackend)
	return lb
}

// GetBackend returns the Lua backend for a session.
// CRC: crc-LuaTrackerAdapter.md | R239
func (a *luaTrackerAdapter) GetBackend(vendedID string) *backend.LuaBackend {
	return a.backend(vendedID)
}

// CreateSession sets the resolver on the session's tracker. The negative-ID counter
// belongs to the backend, so there is nothing else to set up.
// CRC: crc-LuaTrackerAdapter.md | R240
func (a *luaTrackerAdapter) CreateSession(vendedID string, resolver changetracker.Resolver) {
	if lb := a.backend(vendedID); lb != nil {
		lb.GetTracker().Resolver = resolver
	}
}

// DestroySession has nothing to clean up: the session's state goes with the session.
// CRC: crc-LuaTrackerAdapter.md
func (a *luaTrackerAdapter) DestroySession(vendedID string) {}

// GetTracker returns the tracker for a session.
func (a *luaTrackerAdapter) GetTracker(vendedID string) *changetracker.Tracker {
	if lb := a.backend(vendedID); lb != nil {
		return lb.GetTracker()
	}
	return nil
}

// CreateVariable creates a variable in the session's tracker.
// Spec: protocol.md - Server uses positive ID 1 for root, negative IDs for others.
// The root (parentID 0) takes the tracker's auto-assigned ID 1; any other server
// variable takes the session's next negative ID, counted per session.
// CRC: crc-LuaTrackerAdapter.md | R240, R45
func (a *luaTrackerAdapter) CreateVariable(vendedID string, parentID int64, luaObject *gopher.LTable, properties map[string]string) (int64, error) {
	lb := a.backend(vendedID)
	if lb == nil {
		return 0, fmt.Errorf("session %s not found", vendedID)
	}
	tracker := lb.GetTracker()

	if parentID == 0 {
		v := tracker.CreateVariable(luaObject, parentID, "", properties)
		a.config.Log(0, "CREATED ROOT LUA VARIABLE id=%d, type=%s", v.ID, v.Properties["type"])
		lb.TrackVariable(v.ID)
		return v.ID, nil
	}

	id := lb.NextServerVarID()
	v := tracker.CreateVariableWithId(id, luaObject, parentID, "", properties)
	if v == nil {
		return 0, fmt.Errorf("variable ID %d already in use", id)
	}
	a.config.Log(0, "CREATED LUA VARIABLE id=%d, type=%s", id, v.Properties["type"])
	lb.TrackVariable(id)
	return id, nil
}

// variable returns the session's tracker and the variable with this ID in it, or nil.
// CRC: crc-LuaTrackerAdapter.md | R241
func (a *luaTrackerAdapter) variable(vendedID string, id int64) (*changetracker.Tracker, *changetracker.Variable) {
	lb := a.backend(vendedID)
	if lb == nil {
		return nil, nil
	}
	tracker := lb.GetTracker()
	return tracker, tracker.GetVariable(id)
}

// Get retrieves a variable's value and properties in the named session.
// CRC: crc-LuaTrackerAdapter.md | R241
func (a *luaTrackerAdapter) Get(vendedID string, id int64) (json.RawMessage, map[string]string, bool) {
	tracker, v := a.variable(vendedID, id)
	if v == nil {
		return nil, nil, false
	}
	jsonBytes, _ := tracker.ToValueJSONBytes(v.Value)
	return jsonBytes, v.Properties, true
}

// GetProperty retrieves a property value of a variable in the named session.
// CRC: crc-LuaTrackerAdapter.md | R241
func (a *luaTrackerAdapter) GetProperty(vendedID string, id int64, name string) (string, bool) {
	_, v := a.variable(vendedID, id)
	if v == nil {
		return "", false
	}
	val := v.GetProperty(name)
	return val, val != ""
}

// Update is a no-op: the tracker already holds the variable's state. It names its
// session like every other variable operation.
// CRC: crc-LuaTrackerAdapter.md | R241
func (a *luaTrackerAdapter) Update(vendedID string, id int64, value json.RawMessage, properties map[string]string) error {
	return nil
}

// Destroy removes a variable from the named session's tracker.
// CRC: crc-LuaTrackerAdapter.md | R241, R243
func (a *luaTrackerAdapter) Destroy(vendedID string, id int64) error {
	lb := a.backend(vendedID)
	if lb == nil {
		return fmt.Errorf("session %s not found", vendedID)
	}
	lb.GetTracker().DestroyVariable(id)
	lb.UntrackVariable(id)
	return nil
}

// DetectChanges runs change detection for a session.
func (a *luaTrackerAdapter) DetectChanges(vendedID string) bool {
	if lb := a.backend(vendedID); lb != nil {
		return lb.GetTracker().DetectChanges()
	}
	return false
}

// GetChanges returns the detected changes for a session.
func (a *luaTrackerAdapter) GetChanges(vendedID string) []changetracker.Change {
	if lb := a.backend(vendedID); lb != nil {
		return lb.GetTracker().GetChanges()
	}
	return nil
}

// updateVariable1Viewdefs updates variable 1's viewdefs property with new viewdefs.
// Per spec: "pending viewdefs are set on variable 1's viewdefs property"
func (a *luaTrackerAdapter) updateVariable1Viewdefs(vendedID string, viewdefs map[string]string) {
	viewdefsJSON, err := json.Marshal(viewdefs)
	if err != nil {
		a.config.Log(0, "Warning: failed to marshal viewdefs: %v", err)
		return
	}
	if lb := a.backend(vendedID); lb != nil {
		lb.GetTracker().GetVariable(1).SetProperty("viewdefs", string(viewdefsJSON))
	}
}
