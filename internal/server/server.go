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
	"time"

	"github.com/zot/ui/internal/bundle"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/lua"
	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/session"
	"github.com/zot/ui/internal/storage"
	"github.com/zot/ui/internal/variable"
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
// This method looks up the vended ID and calls the Lua runtime's AfterBatch.
func (s *Server) AfterBatch(internalSessionID string) {
	if s.luaRuntime == nil {
		return
	}

	vendedID := s.sessions.GetVendedID(internalSessionID)
	if vendedID == "" {
		return
	}

	// AfterBatch handles change detection internally - updates are sent via store.Update
	// which triggers the normal watch notification mechanism
	s.luaRuntime.AfterBatch(vendedID)
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
type luaStoreAdapter struct {
	store          *variable.Store
	wrapperManager *lua.WrapperManager
}

// SetWrapperManager sets the wrapper manager for creating wrappers during variable creation.
func (a *luaStoreAdapter) SetWrapperManager(wm *lua.WrapperManager) {
	a.wrapperManager = wm
}

func (a *luaStoreAdapter) Create(parentID int64, value json.RawMessage, properties map[string]string) (int64, error) {
	id, err := a.store.Create(variable.CreateOptions{
		ParentID:   parentID,
		Value:      value,
		Properties: properties,
	})
	if err != nil {
		return id, err
	}

	// If wrapper property is set, create wrapper instance
	if wrapperType, ok := properties["wrapper"]; ok && wrapperType != "" && a.wrapperManager != nil {
		v, ok := a.store.Get(id)
		if ok {
			wrapper, err := a.wrapperManager.CreateWrapper(v)
			if err != nil {
				log.Printf("Warning: failed to create wrapper %s for variable %d: %v", wrapperType, id, err)
			} else if wrapper != nil {
				v.SetWrapperInstance(wrapper)
				// Compute initial stored value
				storedValue, err := lua.ComputeStoredValue(wrapper, value)
				if err != nil {
					log.Printf("Warning: failed to compute stored value for variable %d: %v", id, err)
				} else {
					v.SetStoredValue(storedValue)
				}
			}
		}
	}

	return id, nil
}

func (a *luaStoreAdapter) Get(id int64) (json.RawMessage, map[string]string, bool) {
	v, ok := a.store.Get(id)
	if !ok {
		return nil, nil, false
	}
	return v.Value, v.Properties, true
}

func (a *luaStoreAdapter) GetProperty(id int64, name string) (string, bool) {
	v, ok := a.store.Get(id)
	if !ok {
		return "", false
	}
	val, exists := v.Properties[name]
	return val, exists
}

func (a *luaStoreAdapter) Update(id int64, value json.RawMessage, properties map[string]string) error {
	return a.store.Update(id, value, properties)
}

func (a *luaStoreAdapter) Destroy(id int64) error {
	return a.store.Destroy(id)
}
