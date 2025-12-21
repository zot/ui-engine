// CRC: seq-server-startup.md
// Spec: deployment.md
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	gopher "github.com/yuin/gopher-lua"
	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui/internal/backend"
	"github.com/zot/ui/internal/bundle"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/lua"
	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/session"
	"github.com/zot/ui/internal/variable"
	"github.com/zot/ui/internal/viewdef"
)

// Server is the main UI server.
// CRC: crc-Server.md
type Server struct {
	config          *config.Config
	store           *variable.Store
	sessions        *session.Manager
	handler         *protocol.Handler
	pendingQueues   *PendingQueueManager
	httpServer      *http.Server
	httpEndpoint    *HTTPEndpoint
	wsEndpoint      *WebSocketEndpoint
	backendSocket   *BackendSocket
	luaRuntime      *lua.Runtime
	wrapperRegistry *lua.WrapperRegistry
	wrapperManager  *lua.WrapperManager
	storeAdapter    *luaTrackerAdapter
	viewdefManager  *viewdef.ViewdefManager
}

// New creates a new server with the given configuration.
func New(cfg *config.Config) *Server {
	store := variable.NewStore(cfg)

	sessions := session.NewManager(cfg.Session.Timeout.Duration())

	s := &Server{
		config:        cfg,
		store:         store,
		sessions:      sessions,
		pendingQueues: NewPendingQueueManager(),
	}

	// Create message sender that wraps WebSocket endpoint
	sender := &serverMessageSender{server: s}
	s.handler = protocol.NewHandler(cfg, store, sender)

	// Set up pending queue for CLI/REST clients
	s.handler.SetPendingQueuer(s.pendingQueues)

	// Set up backend lookup for per-session watch management
	s.handler.SetBackendLookup(&serverBackendLookup{server: s})

	// Create WebSocket endpoint
	s.wsEndpoint = NewWebSocketEndpoint(cfg, sessions, s.handler)

	// Create HTTP endpoint
	s.httpEndpoint = NewHTTPEndpoint(sessions, s.handler, s.wsEndpoint)

	// Set up site serving (bundle or custom directory)
	s.setupSite(cfg)

	// Set up viewdef manager and load viewdefs
	s.setupViewdefs(cfg)

	// Create backend socket
	s.backendSocket = NewBackendSocket(cfg, cfg.Server.Socket, s.handler, s.httpEndpoint)

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

		// Create wrapper manager with runtime and registry
		s.wrapperManager = lua.NewWrapperManager(s.luaRuntime, s.wrapperRegistry)

		// Set wrapper manager on store adapter so it can create wrappers during variable creation
		if s.storeAdapter != nil {
			s.storeAdapter.SetWrapperManager(s.wrapperManager)
		}

		// Set viewdef manager on store adapter so it can send viewdefs when new types appear
		if s.storeAdapter != nil && s.viewdefManager != nil {
			s.storeAdapter.SetViewdefManager(s.viewdefManager)
		}

		// Note: Go wrappers (like ViewList) auto-register via init() - no explicit registration needed

		// Set session callbacks for Lua session management
		// Callbacks receive vended IDs (compact integers) for backend communication
		// Each session gets its own LuaBackend for per-session watch management
		sessions.SetOnSessionCreated(func(vendedID string, sess *session.Session) error {
			return s.CreateLuaBackendForSession(vendedID, sess)
		})
		sessions.SetOnSessionDestroyed(func(vendedID string, sess *session.Session) {
			s.DestroyLuaBackendForSession(vendedID, sess)
		})

		// Set up afterBatch callback for automatic change detection
		s.wsEndpoint.SetAfterBatch(s.AfterBatch)

		// Set Lua runtime as path variable handler for frontend creates
		s.handler.SetPathVariableHandler(s.luaRuntime)
	}

	return s
}

// Start starts the server.
func (s *Server) Start() error {
	// Start backend socket
	if err := s.backendSocket.Listen(); err != nil {
		return fmt.Errorf("failed to start backend socket: %w", err)
	}
	s.config.Log(0, "Backend socket listening on %s", s.backendSocket.GetSocketPath())

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.httpEndpoint,
	}

	s.config.Log(0, "HTTP server listening on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Shutdown Lua runtime
	if s.luaRuntime != nil {
		s.luaRuntime.Shutdown()
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

// GetStore returns the variable store.
func (s *Server) GetStore() *variable.Store {
	return s.store
}

// GetSessions returns the session manager.
func (s *Server) GetSessions() *session.Manager {
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
		s.httpEndpoint.SetStaticDir(htmlDir)
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
		s.httpEndpoint.SetEmbeddedSite(bundle.NewZipFileSystem(zipReader))
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
		return
	}

	// Try to load from bundle
	if err := s.viewdefManager.LoadFromBundle(); err != nil {
		s.config.Log(0, "Warning: failed to load viewdefs from bundle: %v", err)
	} else if s.viewdefManager.Count() > 0 {
		s.config.Log(0, "Loaded %d viewdefs from bundle", s.viewdefManager.Count())
	}
}

// SetSiteFS sets a custom filesystem for serving static files.
func (s *Server) SetSiteFS(siteFS fs.FS) {
	s.httpEndpoint.SetEmbeddedSite(siteFS)
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

	// Create runtime
	runtime, err := lua.NewRuntime(cfg, luaDir, s.viewdefManager)
	if err != nil {
		s.config.Log(0, "Warning: failed to create Lua runtime: %v", err)
		return
	}

	s.luaRuntime = runtime

	// Create store adapter and set on runtime
	s.storeAdapter = &luaTrackerAdapter{config: cfg, store: s.store, runtime: runtime}
	runtime.SetVariableStore(s.storeAdapter)

	// Set wrapper registry on runtime (allows ui.registerWrapper from Lua)
	runtime.SetWrapperRegistry(s.wrapperRegistry)

	// In bundle mode, pre-cache main.lua content for per-session execution
	// In --dir mode, main.lua is read from filesystem per-session
	if cfg.Server.Dir == "" {
		s.preloadMainLuaFromBundle(runtime)
	}

	s.config.Log(0, "Lua runtime initialized (dir: %s)", luaDir)
}

// CreateLuaBackendForSession creates a LuaBackend for a new session.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
// The backend is attached to the session for per-session watch management.
// CRC: crc-LuaBackend.md
// Sequence: seq-session-create-backend.md
func (s *Server) CreateLuaBackendForSession(vendedID string, sess *session.Session) error {
	if s.luaRuntime == nil {
		return nil // Lua not enabled
	}

	// Create LuaBackend with resolver
	lb := backend.NewLuaBackend(s.config, vendedID, &lua.LuaResolver{}) // Resolver will be set by runtime

	// Attach backend to session
	sess.SetBackend(lb)

	// Also track in store adapter for variable operations
	if s.storeAdapter != nil {
		s.storeAdapter.SetBackend(vendedID, lb)
	}

	// Create Lua session (executes main.lua)
	_, err := s.luaRuntime.CreateLuaSession(vendedID)
	if err != nil {
		sess.SetBackend(nil)
		return err
	}

	return nil
}

// DestroyLuaBackendForSession destroys a session's LuaBackend.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
func (s *Server) DestroyLuaBackendForSession(vendedID string, sess *session.Session) {
	if s.luaRuntime == nil {
		return
	}

	// Clean up Lua runtime state
	s.luaRuntime.DestroyLuaSession(vendedID)

	// Shutdown backend
	if b := sess.GetBackend(); b != nil {
		b.Shutdown()
	}
	sess.SetBackend(nil)

	// Remove from store adapter
	if s.storeAdapter != nil {
		s.storeAdapter.RemoveBackend(vendedID)
	}
}

// AfterBatch triggers Lua change detection after processing a message batch.
// internalSessionID is the full UUID session ID (used in URLs/WebSocket bindings).
// This method looks up the vended ID and calls the Lua runtime's AfterBatch,
// then sends any detected changes to watching frontends.
// CRC: crc-Server.md
// Sequence: seq-relay-message.md, seq-backend-refresh.md
func (s *Server) AfterBatch(internalSessionID string) {
	if s.luaRuntime == nil {
		return
	}

	// Get session and its backend
	sess := s.sessions.Get(internalSessionID)
	if sess == nil {
		return
	}

	b := sess.GetBackend()
	if b == nil {
		return
	}

	vendedID := s.sessions.GetVendedID(internalSessionID)
	if vendedID == "" {
		return
	}

	// Get detected changes from Lua runtime
	updates := s.luaRuntime.AfterBatch(vendedID)
	if len(updates) == 0 {
		return
	}

	// Send updates to watchers (via session's backend)
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

		// Send to all watchers
		for _, connID := range watchers {
			s.wsEndpoint.Send(connID, updateMsg)
		}
	}
}

// GetLuaRuntime returns the Lua runtime (for testing/advanced use).
func (s *Server) GetLuaRuntime() *lua.Runtime {
	return s.luaRuntime
}

// preloadMainLuaFromBundle caches main.lua from bundle if available.
func (s *Server) preloadMainLuaFromBundle(runtime *lua.Runtime) {
	content, err := bundle.ReadFile("lua/main.lua")
	if err != nil {
		// No main.lua in bundle - OK for hybrid/backend-only modes
		return
	}
	runtime.SetMainLuaCode(string(content))
	s.config.Log(0, "Preloaded main.lua from bundle")
}

// luaTrackerAdapter adapts variable.Store to lua.VariableStore interface.
// It coordinates with per-session LuaBackends for change detection.
type luaTrackerAdapter struct {
	config         *config.Config
	store          *variable.Store
	runtime        *lua.Runtime
	wrapperManager *lua.WrapperManager
	viewdefManager *viewdef.ViewdefManager
	backends       map[string]*backend.LuaBackend // vendedID -> backend
	varToSession   map[int64]string               // variableID -> sessionID
	mu             sync.RWMutex
}

// SetWrapperManager sets the wrapper manager for creating wrappers during variable creation.
func (a *luaTrackerAdapter) SetWrapperManager(wm *lua.WrapperManager) {
	a.wrapperManager = wm
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
	}
	a.backends[sessionID] = lb
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

// CreateSession creates a new tracker for a session.
// Note: The tracker is now managed by LuaBackend, this just sets up the resolver.
func (a *luaTrackerAdapter) CreateSession(sessionID string, resolver changetracker.Resolver) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.backends == nil {
		a.backends = make(map[string]*backend.LuaBackend)
		a.varToSession = make(map[int64]string)
	}
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
func (a *luaTrackerAdapter) CreateVariable(sessionID string, parentID int64, luaObject *gopher.LTable, properties map[string]string) (int64, error) {
	a.mu.RLock()
	lb := a.backends[sessionID]
	a.mu.RUnlock()

	if lb == nil {
		return 0, fmt.Errorf("session %s not found", sessionID)
	}
	tracker := lb.GetTracker()

	a.config.Log(0, "CREATING LUA VARIABLE")
	// Create variable in tracker - it handles object registration
	v := tracker.CreateVariable(luaObject, parentID, "", properties)
	id := v.ID

	a.config.Log(0, "created variable, type = %s, changed: %v", v.Properties["type"], tracker.PropertyChanges[v.ID])
	//// Track which session owns this variable
	//a.mu.Lock()
	//a.varToSession[id] = sessionID
	//a.mu.Unlock()

	// Track in backend for cleanup
	lb.TrackVariable(id)

	//// If wrapper property is set, create wrapper instance
	//if wrapperType, ok := properties["wrapper"]; ok && wrapperType != "" && a.wrapperManager != nil {
	//	session, _ := a.runtime.GetLuaSession(sessionID)
	//	session.Log(0, "ERROR: WHY DOES luaTrackerAdapter.CreateVariable need to create the wrapper when the tracker should do it? wrapper: %v", v.WrapperValue)
	//	varad := &lua.TrackerVariableAdapter{Variable: v, Session: session}
	//	//storeVar, ok := a.store.Get(id)
	//	if ok {
	//		wrapper, err := a.wrapperManager.CreateWrapper(sessionID, varad)
	//		if err != nil {
	//			a.config.Log(0, "Warning: failed to create wrapper %s for variable %d: %v", wrapperType, id, err)
	//		} else if wrapper != nil {
	//			varad.WrapperValue = wrapper
	//			varad.WrapperJSON = tracker.ToValueJSON(wrapper)
	//		}
	//	}
	//}

	// Send viewdefs for new types (per spec: viewdefs.md)
	// When a variable is created with a type property, send viewdefs if not already sent for this session
	//if typeName, ok := properties["type"]; ok && typeName != "" && a.viewdefManager != nil {
	//	a.runtime.Log(4, "Sending viewdefs for %s", typeName)
	//	newViewdefs := a.viewdefManager.GetNewViewdefsForSession(sessionID, typeName)
	//	if len(newViewdefs) > 0 {
	//		a.updateVariable1Viewdefs(sessionID, newViewdefs)
	//	}
	//}

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
func (a *luaTrackerAdapter) DetectChanges(sessionID string) []changetracker.Change {
	a.mu.RLock()
	lb := a.backends[sessionID]
	a.mu.RUnlock()

	if lb == nil {
		return nil
	}

	return lb.GetTracker().DetectChanges()
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
