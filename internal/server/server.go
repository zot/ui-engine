// CRC: seq-server-startup.md
// Spec: deployment.md
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui/internal/bundle"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/lua"
	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/session"
	"github.com/zot/ui/internal/storage"
	"github.com/zot/ui/internal/variable"
	gopher "github.com/yuin/gopher-lua"
)

// Server is the main UI server.
type Server struct {
	config          *config.Config
	store           *variable.Store
	watches         *variable.WatchManager
	sessions        *session.Manager
	handler         *protocol.Handler
	pendingQueues   *PendingQueueManager
	storageBackend  storage.Backend
	httpServer      *http.Server
	httpEndpoint    *HTTPEndpoint
	wsEndpoint      *WebSocketEndpoint
	backendSocket   *BackendSocket
	luaRuntime      *lua.Runtime
	wrapperRegistry *lua.WrapperRegistry
	wrapperManager  *lua.WrapperManager
	storeAdapter    *luaStoreAdapter
	viewdefManager  *ViewdefManager
}

// New creates a new server with the given configuration.
func New(cfg *config.Config) *Server {
	store := variable.NewStore()
	watches := variable.NewWatchManager(store)

	sessions := session.NewManager(
		store,
		cfg.Session.Timeout.Duration(),
	)

	s := &Server{
		config:        cfg,
		store:         store,
		watches:       watches,
		sessions:      sessions,
		pendingQueues: NewPendingQueueManager(),
	}

	// Initialize storage backend based on config
	switch cfg.Storage.Type {
	case "sqlite":
		backend, err := storage.NewSQLiteStorage(cfg.Storage.Path)
		if err != nil {
			log.Printf("Warning: failed to initialize SQLite storage: %v", err)
		} else {
			s.storageBackend = backend
			store.SetStorage(backend)
			// Load existing data from storage
			if err := store.LoadFromStorage(); err != nil {
				log.Printf("Warning: failed to load data from storage: %v", err)
			}
		}
	case "memory", "":
		// Memory storage is the default (no persistence)
		backend := storage.NewMemoryStorage()
		s.storageBackend = backend
		store.SetStorage(backend)
	default:
		log.Printf("Warning: unsupported storage type: %s, using memory", cfg.Storage.Type)
	}

	// Create message sender that wraps WebSocket endpoint
	sender := &serverMessageSender{server: s}
	s.handler = protocol.NewHandler(store, watches, sender)

	// Set up pending queue for CLI/REST clients
	s.handler.SetPendingQueuer(s.pendingQueues)

	// Create WebSocket endpoint
	s.wsEndpoint = NewWebSocketEndpoint(sessions, s.handler)

	// Create HTTP endpoint
	s.httpEndpoint = NewHTTPEndpoint(sessions, s.handler, s.wsEndpoint)

	// Set up site serving (bundle or custom directory)
	s.setupSite(cfg)

	// Set up viewdef manager and load viewdefs
	s.setupViewdefs(cfg)

	// Create backend socket
	s.backendSocket = NewBackendSocket(cfg.Server.Socket, s.handler, s.httpEndpoint)

	// Set verbosity on all components
	verbosity := cfg.Verbosity()
	s.wsEndpoint.SetVerbosity(verbosity)
	s.backendSocket.SetVerbosity(verbosity)
	s.handler.SetVerbosity(verbosity)
	store.SetVerbosity(verbosity)

	// Initialize wrapper registry (needed for ViewList wrapper support)
	s.wrapperRegistry = lua.NewWrapperRegistry()

	// Initialize Lua runtime if enabled
	if cfg.Lua.Enabled {
		s.setupLua(cfg, verbosity)

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

		// Wire up watch manager to set active flag on tracker variables
		if s.storeAdapter != nil {
			watches.OnActiveChanged = s.storeAdapter.SetVariableActive
		}

		// Note: Go wrappers (like ViewList) auto-register via init() - no explicit registration needed

		// Set session callbacks for Lua session management
		// Callbacks receive vended IDs (compact integers) for backend communication
		sessions.SetOnSessionCreated(func(vendedID string) error {
			return s.CreateLuaSessionForFrontend(vendedID)
		})
		sessions.SetOnSessionDestroyed(func(vendedID string) {
			s.DestroyLuaSessionForFrontend(vendedID)
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
	log.Printf("Backend socket listening on %s", s.backendSocket.GetSocketPath())

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.httpEndpoint,
	}

	log.Printf("HTTP server listening on %s", addr)
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

	// Close storage backend
	if s.storageBackend != nil {
		s.storageBackend.Close()
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
				log.Printf("Cleaned up %d inactive sessions", count)
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
		log.Printf("Serving site from directory: %s", htmlDir)
		return
	}

	// Try to load from bundle
	zipReader, err := bundle.GetBundleReader()
	if err != nil {
		log.Printf("Warning: failed to read bundle: %v", err)
		return
	}

	if zipReader != nil {
		// NewZipFileSystem automatically serves from html/ subdirectory
		s.httpEndpoint.SetEmbeddedSite(bundle.NewZipFileSystem(zipReader))
		log.Printf("Serving site from embedded bundle (html/)")
		return
	}

	log.Printf("Warning: no site available (not bundled and no --dir specified)")
}

// setupViewdefs initializes the viewdef manager and loads viewdefs.
func (s *Server) setupViewdefs(cfg *config.Config) {
	s.viewdefManager = NewViewdefManager()

	// If --dir is specified, load from that directory's viewdefs/ subdirectory
	if cfg.Server.Dir != "" {
		viewdefsDir := cfg.Server.Dir + "/viewdefs"
		if err := s.viewdefManager.LoadFromDirectory(viewdefsDir); err != nil {
			log.Printf("Warning: failed to load viewdefs from %s: %v", viewdefsDir, err)
		} else {
			log.Printf("Loaded %d viewdefs from directory: %s", s.viewdefManager.Count(), viewdefsDir)
		}
		return
	}

	// Try to load from bundle
	if err := s.viewdefManager.LoadFromBundle(); err != nil {
		log.Printf("Warning: failed to load viewdefs from bundle: %v", err)
	} else if s.viewdefManager.Count() > 0 {
		log.Printf("Loaded %d viewdefs from bundle", s.viewdefManager.Count())
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

// setupLua initializes the Lua runtime.
func (s *Server) setupLua(cfg *config.Config, verbosity int) {
	// Determine Lua directory
	luaDir := cfg.Lua.Path
	if cfg.Server.Dir != "" {
		luaDir = filepath.Join(cfg.Server.Dir, "lua")
	}

	// Create runtime
	runtime, err := lua.NewRuntime(luaDir)
	if err != nil {
		log.Printf("Warning: failed to create Lua runtime: %v", err)
		return
	}

	runtime.SetVerbosity(verbosity)
	s.luaRuntime = runtime

	// Create store adapter and set on runtime
	s.storeAdapter = &luaStoreAdapter{store: s.store}
	runtime.SetVariableStore(s.storeAdapter)

	// Set wrapper registry on runtime (allows ui.registerWrapper from Lua)
	runtime.SetWrapperRegistry(s.wrapperRegistry)

	// Pre-load main.lua from bundle if available (for bundle mode)
	// When sessions connect, this will be executed per-session
	s.preloadMainLuaFromBundle(runtime)

	// Auto-load library Lua files (not main.lua - that's loaded per session)
	// These are shared libraries available to all sessions
	if err := runtime.LoadDirectory(luaDir); err != nil {
		// Try loading libraries from bundle
		s.loadLuaLibrariesFromBundle(runtime)
	}

	log.Printf("Lua runtime initialized (dir: %s)", luaDir)
}

// CreateLuaSessionForFrontend creates a Lua session for a new frontend session.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
func (s *Server) CreateLuaSessionForFrontend(vendedID string) error {
	if s.luaRuntime == nil {
		return nil // Lua not enabled
	}
	_, err := s.luaRuntime.CreateLuaSession(vendedID)
	return err
}

// DestroyLuaSessionForFrontend destroys a Lua session.
// vendedID is the compact integer ID (e.g., "1", "2") for backend communication.
func (s *Server) DestroyLuaSessionForFrontend(vendedID string) {
	if s.luaRuntime == nil {
		return
	}
	s.luaRuntime.DestroyLuaSession(vendedID)
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

	vendedID := s.sessions.GetVendedID(internalSessionID)
	if vendedID == "" {
		return
	}

	// Get detected changes from Lua runtime
	updates := s.luaRuntime.AfterBatch(vendedID)
	if len(updates) == 0 {
		return
	}

	// Send updates to watchers
	for _, update := range updates {
		watchers := s.watches.GetWatchers(update.VarID)
		if len(watchers) == 0 {
			continue
		}

		// Build update message
		updateMsg, err := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
			VarID: update.VarID,
			Value: update.Value,
		})
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
	log.Printf("Preloaded main.lua from bundle")
}

// loadLuaLibrariesFromBundle loads library Lua files from embedded bundle.
// Skips main.lua which is loaded per-session.
func (s *Server) loadLuaLibrariesFromBundle(runtime *lua.Runtime) {
	files, err := bundle.ListFilesInDir("lua")
	if err != nil {
		// No bundle or no lua directory in bundle
		return
	}

	loaded := 0
	for _, filePath := range files {
		if !strings.HasSuffix(filePath, ".lua") {
			continue
		}
		// Skip backup/temp files
		filename := filepath.Base(filePath)
		if strings.HasPrefix(filename, ".") {
			continue
		}
		// Skip main.lua - loaded per-session
		if filename == "main.lua" {
			continue
		}

		content, err := bundle.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: failed to read bundled Lua file %s: %v", filePath, err)
			continue
		}

		if err := runtime.LoadCode(filename, string(content)); err != nil {
			log.Printf("Warning: failed to load bundled Lua file %s: %v", filename, err)
		} else {
			loaded++
			log.Printf("Loaded bundled Lua library: %s", filename)
		}
	}

	if loaded > 0 {
		log.Printf("Loaded %d Lua library files from bundle", loaded)
	}
}

// luaStoreAdapter adapts variable.Store to lua.VariableStore interface.
// It holds a tracker per session for change detection.
type luaStoreAdapter struct {
	store          *variable.Store
	wrapperManager *lua.WrapperManager
	viewdefManager *ViewdefManager
	trackers       map[string]*changetracker.Tracker // sessionID -> tracker
	varToSession   map[int64]string                  // variableID -> sessionID
	mu             sync.RWMutex
}

// SetWrapperManager sets the wrapper manager for creating wrappers during variable creation.
func (a *luaStoreAdapter) SetWrapperManager(wm *lua.WrapperManager) {
	a.wrapperManager = wm
}

// SetViewdefManager sets the viewdef manager for sending viewdefs to frontend.
func (a *luaStoreAdapter) SetViewdefManager(vm *ViewdefManager) {
	a.viewdefManager = vm
}

// CreateSession creates a new tracker for a session.
func (a *luaStoreAdapter) CreateSession(sessionID string, resolver changetracker.Resolver) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.trackers == nil {
		a.trackers = make(map[string]*changetracker.Tracker)
		a.varToSession = make(map[int64]string)
	}
	tracker := changetracker.NewTracker()
	tracker.Resolver = resolver
	a.trackers[sessionID] = tracker
}

// DestroySession removes a session's tracker.
func (a *luaStoreAdapter) DestroySession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.trackers, sessionID)
	// Clean up varToSession
	for varID, sid := range a.varToSession {
		if sid == sessionID {
			delete(a.varToSession, varID)
		}
	}
}

// GetTracker returns the tracker for a session.
func (a *luaStoreAdapter) GetTracker(sessionID string) *changetracker.Tracker {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.trackers[sessionID]
}

// CreateVariable creates a variable using the session's tracker.
func (a *luaStoreAdapter) CreateVariable(sessionID string, parentID int64, luaObject *gopher.LTable, properties map[string]string) (int64, error) {
	a.mu.RLock()
	tracker := a.trackers[sessionID]
	a.mu.RUnlock()

	if tracker == nil {
		return 0, fmt.Errorf("session %s not found", sessionID)
	}

	// Create variable in tracker - it handles object registration
	v := tracker.CreateVariable(luaObject, parentID, "", properties)
	id := v.ID

	// Track which session owns this variable
	a.mu.Lock()
	a.varToSession[id] = sessionID
	a.mu.Unlock()

	// Also create in store for persistence and property watchers
	// Use same ID as tracker to keep them in sync
	jsonValue, _ := tracker.ToValueJSONBytes(luaObject)
	_, err := a.store.Create(variable.CreateOptions{
		ID:         id,
		ParentID:   parentID,
		Value:      jsonValue,
		Properties: properties,
	})
	if err != nil {
		log.Printf("Warning: failed to create variable %d in store: %v", id, err)
	}

	// If wrapper property is set, create wrapper instance
	if wrapperType, ok := properties["wrapper"]; ok && wrapperType != "" && a.wrapperManager != nil {
		storeVar, ok := a.store.Get(id)
		if ok {
			wrapper, err := a.wrapperManager.CreateWrapper(storeVar)
			if err != nil {
				log.Printf("Warning: failed to create wrapper %s for variable %d: %v", wrapperType, id, err)
			} else if wrapper != nil {
				storeVar.SetWrapperInstance(wrapper)
				storedValue, err := lua.ComputeStoredValue(wrapper, jsonValue)
				if err != nil {
					log.Printf("Warning: failed to compute stored value for variable %d: %v", id, err)
				} else {
					storeVar.SetStoredValue(storedValue)
				}
			}
		}
	}

	// Send viewdefs for new types (per spec: viewdefs.md)
	// When a variable is created with a type property, send viewdefs if not already sent for this session
	if typeName, ok := properties["type"]; ok && typeName != "" && a.viewdefManager != nil {
		newViewdefs := a.viewdefManager.GetNewViewdefsForSession(sessionID, typeName)
		if len(newViewdefs) > 0 {
			a.updateVariable1Viewdefs(sessionID, newViewdefs)
		}
	}

	return id, nil
}

// CreatePathVariable creates a path-based variable initiated by the frontend.
// This is called when the frontend creates a variable with parentId and path property.
// The variable is created in the parent's tracker, which resolves the path.
func (a *luaStoreAdapter) CreatePathVariable(parentID int64, path string, properties map[string]string) (int64, json.RawMessage, error) {
	// Find which session owns the parent variable
	a.mu.RLock()
	sessionID, ok := a.varToSession[parentID]
	if !ok {
		a.mu.RUnlock()
		return 0, nil, fmt.Errorf("parent variable %d not found in any session", parentID)
	}
	tracker := a.trackers[sessionID]
	a.mu.RUnlock()

	if tracker == nil {
		return 0, nil, fmt.Errorf("session %s tracker not found", sessionID)
	}

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

	// Also create in store for persistence
	_, err = a.store.Create(variable.CreateOptions{
		ID:         id,
		ParentID:   parentID,
		Value:      jsonValue,
		Properties: properties,
	})
	if err != nil {
		log.Printf("Warning: failed to create path variable %d in store: %v", id, err)
	}

	return id, jsonValue, nil
}

// Get retrieves a variable's value and properties.
func (a *luaStoreAdapter) Get(id int64) (json.RawMessage, map[string]string, bool) {
	// First try the tracker
	a.mu.RLock()
	sessionID, ok := a.varToSession[id]
	if ok {
		tracker := a.trackers[sessionID]
		a.mu.RUnlock()
		if tracker != nil {
			v := tracker.GetVariable(id)
			if v != nil {
				jsonBytes, _ := tracker.ToValueJSONBytes(v.Value)
				return jsonBytes, v.Properties, true
			}
		}
	} else {
		a.mu.RUnlock()
	}

	// Fall back to store
	v, ok := a.store.Get(id)
	if !ok {
		return nil, nil, false
	}
	return v.Value, v.Properties, true
}

// GetProperty retrieves a property value.
func (a *luaStoreAdapter) GetProperty(id int64, name string) (string, bool) {
	// First try the tracker
	a.mu.RLock()
	sessionID, ok := a.varToSession[id]
	if ok {
		tracker := a.trackers[sessionID]
		a.mu.RUnlock()
		if tracker != nil {
			v := tracker.GetVariable(id)
			if v != nil {
				val := v.GetProperty(name)
				return val, val != ""
			}
		}
	} else {
		a.mu.RUnlock()
	}

	// Fall back to store
	v, ok := a.store.Get(id)
	if !ok {
		return "", false
	}
	val, exists := v.Properties[name]
	return val, exists
}

// Update updates a variable's value and/or properties in the store.
func (a *luaStoreAdapter) Update(id int64, value json.RawMessage, properties map[string]string) error {
	return a.store.Update(id, value, properties)
}

// Destroy removes a variable.
func (a *luaStoreAdapter) Destroy(id int64) error {
	// Remove from tracker
	a.mu.Lock()
	sessionID, ok := a.varToSession[id]
	if ok {
		if tracker := a.trackers[sessionID]; tracker != nil {
			tracker.DestroyVariable(id)
		}
		delete(a.varToSession, id)
	}
	a.mu.Unlock()

	// Remove from store
	return a.store.Destroy(id)
}

// DetectChanges returns changes for a session.
func (a *luaStoreAdapter) DetectChanges(sessionID string) []changetracker.Change {
	a.mu.RLock()
	tracker := a.trackers[sessionID]
	a.mu.RUnlock()

	if tracker == nil {
		return nil
	}

	return tracker.DetectChanges()
}

// SetVariableActive sets the active flag on a tracker variable.
// Called by WatchManager on watch/unwatch transitions.
func (a *luaStoreAdapter) SetVariableActive(varID int64, active bool) {
	a.mu.RLock()
	sessionID, ok := a.varToSession[varID]
	if !ok {
		a.mu.RUnlock()
		return
	}
	tracker := a.trackers[sessionID]
	a.mu.RUnlock()

	if tracker == nil {
		return
	}

	v := tracker.GetVariable(varID)
	if v != nil {
		v.SetActive(active)
	}
}

// updateVariable1Viewdefs updates variable 1's viewdefs property with new viewdefs.
// Per spec: "pending viewdefs are set on variable 1's viewdefs property"
func (a *luaStoreAdapter) updateVariable1Viewdefs(sessionID string, viewdefs map[string]string) {
	// Serialize viewdefs as JSON
	viewdefsJSON, err := json.Marshal(viewdefs)
	if err != nil {
		log.Printf("Warning: failed to marshal viewdefs: %v", err)
		return
	}

	// Update variable 1's viewdefs property
	// Variable 1 is always the app variable (first variable created per session)
	err = a.store.Update(1, nil, map[string]string{
		"viewdefs": string(viewdefsJSON),
	})
	if err != nil {
		log.Printf("Warning: failed to update variable 1 viewdefs property: %v", err)
	}
}
