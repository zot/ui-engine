// CRC: crc-LuaSession.md (Server owns luaSessions map)
// Spec: deployment.md, interfaces.md
// Sequence: seq-server-startup.md, seq-session-create-backend.md, seq-lua-session-init.md
//
// Server implements per-session Lua isolation via luaSessions map[string]*lua.LuaSession.
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
	luaSessions      map[string]*lua.LuaSession // vendedID -> per-session Lua runtime
	luaSessionsMu    sync.RWMutex
	luaConfig        *luaSetupConfig // Shared config for creating new sessions
	wrapperRegistry  *lua.WrapperRegistry
	storeAdapter     *luaTrackerAdapter
	viewdefManager   *viewdef.ViewdefManager
	hotLoader        *lua.HotLoader     // Lua hot-reloading (nil if disabled)
	viewdefHotLoader *viewdef.HotLoader // Viewdef hot-reloading (nil if disabled)
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
			s.luaSessionsMu.RLock()
			luaSession := s.luaSessions[sessionID]
			s.luaSessionsMu.RUnlock()
			if luaSession == nil {
				return nil, 0, fmt.Errorf("session %s not found", sessionID)
			}
			tracker := luaSession.GetTracker()
			if tracker == nil {
				return nil, 0, fmt.Errorf("tracker not found")
			}
			// CRC: crc-HTTPEndpoint.md (R62)
			if diagLevel > 0 {
				tracker.DiagLevel = diagLevel
				defer func() { tracker.DiagLevel = 0 }()
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
// Returns the server, listener, and base URL.
func (s *Server) configureHTTP(port int) (*http.Server, net.Listener, string, error) {
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

	// Shutdown all Lua sessions
	s.luaSessionsMu.Lock()
	for vendedID, luaSession := range s.luaSessions {
		s.config.Log(0, "Shutting down Lua session %s", vendedID)
		luaSession.Shutdown()
	}
	s.luaSessions = nil
	s.luaSessionsMu.Unlock()

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

// setupViewdefs initializes the viewdef manager and loads viewdefs.
func (s *Server) setupViewdefs(cfg *config.Config) {
	s.viewdefManager = viewdef.NewViewdefManager()

	// If --dir is specified, load from that directory's viewdefs/ subdirectory
	if cfg.Server.Dir != "" {
		viewdefsDir := cfg.Server.Dir + "/viewdefs"
		if err := s.viewdefManager.LoadFromDirectory(viewdefsDir); err != nil {
			s.config.Log(0, "Warning: failed to load viewdefs from %s: %v", viewdefsDir, err)
		} else {
			s.config.Log(0, "Loaded %d viewdefs from directory: %s", s.viewdefManager.Count(), viewdefsDir)
		}

		// Initialize viewdef hot-loader if Lua hot-loading is enabled
		// (reuse the same config flag for viewdefs)
		if cfg.Lua.Hotload {
			s.setupViewdefHotLoader(cfg, viewdefsDir)
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
func (s *Server) setupViewdefHotLoader(cfg *config.Config, viewdefsDir string) {
	hotLoader, err := viewdef.NewHotLoader(
		cfg,
		viewdefsDir,
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

	s.config.Log(0, "ViewdefHotLoader: watching %s for changes", viewdefsDir)
}

// GetSessionIDs returns all active vended session IDs.
// Implements viewdef.SessionPusher.
func (s *Server) GetSessionIDs() []string {
	s.luaSessionsMu.RLock()
	defer s.luaSessionsMu.RUnlock()

	ids := make([]string, 0, len(s.luaSessions))
	for vendedID := range s.luaSessions {
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

	// Initialize sessions map and shared config
	s.luaSessions = make(map[string]*lua.LuaSession)
	s.luaConfig = &luaSetupConfig{
		config: cfg,
		luaDir: luaDir,
	}

	// Create store adapter (will be shared across sessions)
	s.storeAdapter = &luaTrackerAdapter{config: cfg} //, store: s.store}

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

	// Create LuaBackend with resolver
	lb := backend.NewLuaBackend(s.config, vendedID, &lua.LuaResolver{})

	// Attach backend to session
	sess.SetBackend(lb)

	// Create per-session outgoing batcher
	// Each session has its own batcher for isolated debouncing
	sess.SetBatcher(NewOutgoingBatcher(s.wsEndpoint))

	// Track in store adapter for variable operations
	if s.storeAdapter != nil {
		s.storeAdapter.SetBackend(vendedID, lb)
		s.storeAdapter.SetLuaSession(vendedID, luaSession)
	}

	// Set variable store on Lua session
	luaSession.SetVariableStore(s.storeAdapter)

	// Store in our sessions map
	s.luaSessionsMu.Lock()
	s.luaSessions[vendedID] = luaSession
	s.luaSessionsMu.Unlock()

	// Initialize the session (creates session table, runs main.lua)
	_, err = luaSession.CreateLuaSession(vendedID)
	if err != nil {
		s.luaSessionsMu.Lock()
		delete(s.luaSessions, vendedID)
		s.luaSessionsMu.Unlock()
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

	// Get and remove the Lua session
	s.luaSessionsMu.Lock()
	luaSession := s.luaSessions[vendedID]
	delete(s.luaSessions, vendedID)
	s.luaSessionsMu.Unlock()

	// Shutdown the Lua session (closes Lua state)
	if luaSession != nil {
		luaSession.Shutdown()
	}

	// Shutdown backend
	if b := sess.GetBackend(); b != nil {
		b.Shutdown()
	}
	sess.SetBackend(nil)

	// Remove from store adapter
	if s.storeAdapter != nil {
		s.storeAdapter.RemoveBackend(vendedID)
		s.storeAdapter.RemoveLuaSession(vendedID)
	}

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

	// Look up the Lua session for this vended ID
	s.luaSessionsMu.RLock()
	luaSession := s.luaSessions[vendedID]
	s.luaSessionsMu.RUnlock()
	if luaSession == nil {
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

// ExecuteInSession executes code within a session's context.
// This queues through the session's executor to serialize with WebSocket operations.
// AfterBatch is called after execution to detect and push any changes.
// Also sets up the Lua session context so session:getApp() etc. work.
// vendedID is the compact session ID ("1", "2", etc.)
func (s *Server) ExecuteInSession(vendedID string, fn func() (interface{}, error)) (interface{}, error) {
	internalID := s.sessions.GetInternalID(vendedID)
	if internalID == "" {
		return nil, fmt.Errorf("session %s not found", vendedID)
	}

	// Look up the Lua session for this vended ID
	s.luaSessionsMu.RLock()
	luaSession := s.luaSessions[vendedID]
	s.luaSessionsMu.RUnlock()
	if luaSession == nil {
		return nil, fmt.Errorf("Lua session %s not found", vendedID)
	}

	// Delegate to websocket endpoint (queues through session's executor)
	// Wrap fn to set up Lua session context
	return s.wsEndpoint.ExecuteInSession(internalID, func() (interface{}, error) {
		return luaSession.ExecuteInSession(vendedID, fn)
	})
}

// GetLuaSession returns a Lua session by vended ID (for testing/advanced use).
func (s *Server) GetLuaSession(vendedID string) *lua.LuaSession {
	s.luaSessionsMu.RLock()
	defer s.luaSessionsMu.RUnlock()
	return s.luaSessions[vendedID]
}

// getLuaSessions returns all active Lua sessions (used by HotLoader).
func (s *Server) getLuaSessions() []*lua.LuaSession {
	s.luaSessionsMu.RLock()
	defer s.luaSessionsMu.RUnlock()
	sessions := make([]*lua.LuaSession, 0, len(s.luaSessions))
	for _, sess := range s.luaSessions {
		sessions = append(sessions, sess)
	}
	return sessions
}

// HandleFrontendCreate implements PathVariableHandler.
// It delegates to the per-session LuaSession.
// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
func (s *Server) HandleFrontendCreate(sessionID string, id int64, parentID int64, properties map[string]string) error {
	s.luaSessionsMu.RLock()
	luaSession := s.luaSessions[sessionID]
	s.luaSessionsMu.RUnlock()
	if luaSession == nil {
		return fmt.Errorf("Lua session %s not found", sessionID)
	}
	return luaSession.HandleFrontendCreate(sessionID, id, parentID, properties)
}

// HandleFrontendUpdate implements PathVariableHandler.
// It delegates to the per-session LuaSession.
func (s *Server) HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage, properties map[string]string) error {
	s.luaSessionsMu.RLock()
	luaSession := s.luaSessions[sessionID]
	s.luaSessionsMu.RUnlock()
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

// luaTrackerAdapter adapts variable.Store to lua.VariableStore interface.
// It coordinates with per-session LuaBackends for change detection.
type luaTrackerAdapter struct {
	config          *config.Config
	viewdefManager  *viewdef.ViewdefManager
	backends        map[string]*backend.LuaBackend // vendedID -> backend
	luaSessions     map[string]*lua.LuaSession     // vendedID -> LuaSession
	varToSession    map[int64]string               // variableID -> sessionID
	nextServerVarId map[string]int64               // vendedID -> next negative ID (starts at -1, decrements)
	mu              sync.RWMutex
}

// SetViewdefManager sets the viewdef manager for sending viewdefs to frontend.
func (a *luaTrackerAdapter) SetViewdefManager(vm *viewdef.ViewdefManager) {
	a.viewdefManager = vm
}

// SetBackend sets the backend for a session.
func (a *luaTrackerAdapter) SetBackend(sessionID string, lb *backend.LuaBackend) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backends == nil {
		a.backends = make(map[string]*backend.LuaBackend)
		a.varToSession = make(map[int64]string)
		a.nextServerVarId = make(map[string]int64)
	}
	a.backends[sessionID] = lb
}

// GetBackend returns the backend for a session.
func (a *luaTrackerAdapter) GetBackend(sessionID string) *backend.LuaBackend {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.backends[sessionID]
}

// RemoveBackend removes the backend for a session.
func (a *luaTrackerAdapter) RemoveBackend(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.backends, sessionID)
	// Clean up varToSession
	for varID, sid := range a.varToSession {
		if sid == sessionID {
			delete(a.varToSession, varID)
		}
	}
}

// SetLuaSession sets the LuaSession for a session.
func (a *luaTrackerAdapter) SetLuaSession(sessionID string, ls *lua.LuaSession) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.luaSessions == nil {
		a.luaSessions = make(map[string]*lua.LuaSession)
	}
	a.luaSessions[sessionID] = ls
}

// RemoveLuaSession removes the LuaSession for a session.
func (a *luaTrackerAdapter) RemoveLuaSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.luaSessions, sessionID)
}

// CreateSession creates a new tracker for a session.
// Note: The tracker is now managed by LuaBackend, this just sets up the resolver.
func (a *luaTrackerAdapter) CreateSession(sessionID string, resolver changetracker.Resolver) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backends == nil {
		a.backends = make(map[string]*backend.LuaBackend)
	}
	if a.varToSession == nil {
		a.varToSession = make(map[int64]string)
	}
	if a.nextServerVarId == nil {
		a.nextServerVarId = make(map[string]int64)
	}
	// Initialize negative ID counter for this session (starts at -1, decrements)
	// Spec: protocol.md - Server-created variables (other than root) use negative IDs
	a.nextServerVarId[sessionID] = -1
	// If we have a backend for this session, set its resolver
	if lb, ok := a.backends[sessionID]; ok {
		lb.GetTracker().Resolver = resolver
	}
}

// DestroySession removes a session's tracker.
func (a *luaTrackerAdapter) DestroySession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Backend removal is handled separately via RemoveBackend
	// Just clean up varToSession
	for varID, sid := range a.varToSession {
		if sid == sessionID {
			delete(a.varToSession, varID)
		}
	}
}

// GetTracker returns the tracker for a session.
func (a *luaTrackerAdapter) GetTracker(sessionID string) *changetracker.Tracker {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if lb, ok := a.backends[sessionID]; ok {
		return lb.GetTracker()
	}
	return nil
}

// CreateVariable creates a variable using the session's tracker.
// Spec: protocol.md - Server uses positive ID 1 for root, negative IDs for others.
// Root variable (parentID == 0) uses auto-assigned ID (1).
// Non-root server variables use negative IDs starting from -1.
func (a *luaTrackerAdapter) CreateVariable(sessionID string, parentID int64, luaObject *gopher.LTable, properties map[string]string) (int64, error) {
	a.mu.Lock()
	lb := a.backends[sessionID]
	if lb == nil {
		a.mu.Unlock()
		return 0, fmt.Errorf("session %s not found", sessionID)
	}
	tracker := lb.GetTracker()

	var v *changetracker.Variable
	var id int64

	if parentID == 0 {
		// Root variable - use auto-assigned ID (will be 1)
		v = tracker.CreateVariable(luaObject, parentID, "", properties)
		id = v.ID
	} else {
		// Non-root server variable - use negative ID
		id = a.nextServerVarId[sessionID]
		a.nextServerVarId[sessionID]--
		a.mu.Unlock()

		v = tracker.CreateVariableWithId(id, luaObject, parentID, "", properties)
		if v == nil {
			return 0, fmt.Errorf("variable ID %d already in use", id)
		}

		a.config.Log(0, "CREATED LUA VARIABLE id=%d, type=%s", id, v.Properties["type"])
		lb.TrackVariable(id)
		return id, nil
	}
	a.mu.Unlock()

	a.config.Log(0, "CREATED ROOT LUA VARIABLE id=%d, type=%s", id, v.Properties["type"])
	lb.TrackVariable(id)
	return id, nil
}

// CreatePathVariable creates a path-based variable initiated by the frontend.
// This is called when the frontend creates a variable with parentId and path property.
// The variable is created in the parent's tracker, which resolves the path.
func (a *luaTrackerAdapter) CreatePathVariable(parentID int64, path string, properties map[string]string) (int64, json.RawMessage, error) {
	// Find which session owns the parent variable
	a.mu.RLock()
	sessionID, ok := a.varToSession[parentID]
	if !ok {
		a.mu.RUnlock()
		return 0, nil, fmt.Errorf("parent variable %d not found in any session", parentID)
	}
	lb := a.backends[sessionID]
	a.mu.RUnlock()

	if lb == nil {
		return 0, nil, fmt.Errorf("session %s backend not found", sessionID)
	}
	tracker := lb.GetTracker()

	// Ensure path is in properties
	if properties == nil {
		properties = make(map[string]string)
	}
	properties["path"] = path

	// Create variable in tracker - it will resolve the path
	v := tracker.CreateVariable(nil, parentID, path, properties)
	id := v.ID

	// Track which session owns this variable
	a.mu.Lock()
	a.varToSession[id] = sessionID
	a.mu.Unlock()

	// Get the resolved value
	resolvedValue, err := v.Get()
	if err != nil {
		// Path resolution failed - return with nil value, error will be sent as update
		return id, nil, nil
	}

	// Convert to JSON
	jsonValue, err := tracker.ToValueJSONBytes(resolvedValue)
	if err != nil {
		return id, nil, nil
	}

	return id, jsonValue, nil
}

// Get retrieves a variable's value and properties.
func (a *luaTrackerAdapter) Get(id int64) (json.RawMessage, map[string]string, bool) {
	// First try the backend's tracker
	a.mu.RLock()
	sessionID, ok := a.varToSession[id]
	if ok {
		lb := a.backends[sessionID]
		a.mu.RUnlock()
		if lb != nil {
			tracker := lb.GetTracker()
			v := tracker.GetVariable(id)
			if v != nil {
				jsonBytes, _ := tracker.ToValueJSONBytes(v.Value)
				return jsonBytes, v.Properties, true
			}
		}
	} else {
		a.mu.RUnlock()
	}
	return nil, nil, false
}

// GetProperty retrieves a property value.
func (a *luaTrackerAdapter) GetProperty(id int64, name string) (string, bool) {
	// First try the backend's tracker
	a.mu.RLock()
	sessionID, ok := a.varToSession[id]
	if ok {
		lb := a.backends[sessionID]
		a.mu.RUnlock()
		if lb != nil {
			tracker := lb.GetTracker()
			v := tracker.GetVariable(id)
			if v != nil {
				val := v.GetProperty(name)
				return val, val != ""
			}
		}
	} else {
		a.mu.RUnlock()
	}
	return "", false
}

// Update updates a variable's value and/or properties in the store.
func (a *luaTrackerAdapter) Update(id int64, value json.RawMessage, properties map[string]string) error {
	return nil
}

// Destroy removes a variable.
func (a *luaTrackerAdapter) Destroy(id int64) error {
	// Remove from backend's tracker
	a.mu.Lock()
	sessionID, ok := a.varToSession[id]
	if ok {
		if lb := a.backends[sessionID]; lb != nil {
			lb.GetTracker().DestroyVariable(id)
			lb.UntrackVariable(id)
		}
		delete(a.varToSession, id)
	}
	a.mu.Unlock()
	return nil
}

// DetectChanges returns changes for a session.
func (a *luaTrackerAdapter) DetectChanges(sessionID string) bool {
	a.mu.RLock()
	lb := a.backends[sessionID]
	a.mu.RUnlock()
	if lb != nil {
		return lb.GetTracker().DetectChanges()
	}
	return false
}

// DetectChanges returns changes for a session.
func (a *luaTrackerAdapter) GetChanges(sessionID string) []changetracker.Change {
	a.mu.RLock()
	lb := a.backends[sessionID]
	a.mu.RUnlock()

	if lb == nil {
		return nil
	}

	return lb.GetTracker().GetChanges()
}

// updateVariable1Viewdefs updates variable 1's viewdefs property with new viewdefs.
// Per spec: "pending viewdefs are set on variable 1's viewdefs property"
func (a *luaTrackerAdapter) updateVariable1Viewdefs(sessionID string, viewdefs map[string]string) {
	// Serialize viewdefs as JSON
	viewdefsJSON, err := json.Marshal(viewdefs)
	if err != nil {
		a.config.Log(0, "Warning: failed to marshal viewdefs: %v", err)
		return
	}

	lb := a.backends[sessionID]
	lb.GetTracker().GetVariable(1).SetProperty("viewdefs", string(viewdefsJSON))
}
