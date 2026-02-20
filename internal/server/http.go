// CRC: crc-HTTPEndpoint.md
// Spec: interfaces.md, deployment.md
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	changetracker "github.com/zot/change-tracker"
	"github.com/zot/ui-engine/internal/lua"
	"github.com/zot/ui-engine/internal/protocol"
)

// DebugDataProvider is called to get variable data for the debug page.
// Returns variables, tracker change count, and error.
type DebugDataProvider func(sessionID string, diagLevel int) ([]DebugVariable, int64, error)

// RootSessionProvider returns the session ID to use for the root path "/".
// If it returns an empty string, the default behavior (create new session and redirect) is used.
// If it returns a session ID, index.html is served with a session cookie set.
type RootSessionProvider func() string

// DebugVariable represents a variable for the debug tree view.
// CRC: crc-HTTPEndpoint.md (R57, R59, R60, R61)
type DebugVariable struct {
	Session        *lua.LuaSession         `json:"-"`
	Tracker        *changetracker.Tracker  `json:"-"`
	Variable       *changetracker.Variable `json:"-"`
	ID             int64                   `json:"id"`
	ParentID       int64                   `json:"parentId"`
	Type           string                  `json:"type,omitempty"`
	GoType         string                  `json:"goType,omitempty"`
	Path           string                  `json:"path,omitempty"`
	Value          any                     `json:"value,omitempty"`
	BaseValue      any                     `json:"baseValue,omitempty"`
	Properties     map[string]string       `json:"properties,omitempty"`
	ChildIDs       []int64                 `json:"childIds,omitempty"`
	Error          string                  `json:"error,omitempty"`
	ComputeTime    string                  `json:"computeTime,omitempty"`
	MaxComputeTime string                  `json:"maxComputeTime,omitempty"`
	Active         bool                    `json:"active"`
	Access         string                  `json:"access,omitempty"`
	Diags          []string                `json:"diags,omitempty"`
	ChangeCount    int64                   `json:"changeCount"`
	Depth          int                     `json:"depth"`
}

// HTTPEndpoint handles HTTP requests.
type HTTPEndpoint struct {
	sessions            *SessionManager
	handler             *protocol.Handler
	wsEndpoint          *WebSocketEndpoint
	staticDir           string
	embeddedSite        fs.FS
	mux                 *http.ServeMux
	debugDataProvider   DebugDataProvider
	rootSessionProvider RootSessionProvider
}

// NewHTTPEndpoint creates a new HTTP endpoint.
func NewHTTPEndpoint(sessions *SessionManager, handler *protocol.Handler, wsEndpoint *WebSocketEndpoint) *HTTPEndpoint {
	h := &HTTPEndpoint{
		sessions:   sessions,
		handler:    handler,
		wsEndpoint: wsEndpoint,
		mux:        http.NewServeMux(),
	}
	h.setupRoutes()
	return h
}

// SetStaticDir sets a custom directory for static files.
func (h *HTTPEndpoint) SetStaticDir(dir string) {
	h.staticDir = dir
}

// SetEmbeddedSite sets the embedded site filesystem.
func (h *HTTPEndpoint) SetEmbeddedSite(site fs.FS) {
	h.embeddedSite = site
}

// SetDebugDataProvider sets the callback for getting debug variable data.
func (h *HTTPEndpoint) SetDebugDataProvider(provider DebugDataProvider) {
	h.debugDataProvider = provider
}

// SetRootSessionProvider sets a provider for the root path "/" session.
// If the provider returns a session ID, that session is used instead of creating a new one.
func (h *HTTPEndpoint) SetRootSessionProvider(provider RootSessionProvider) {
	h.rootSessionProvider = provider
}

// HandleFunc registers a custom handler on the HTTP mux.
func (h *HTTPEndpoint) HandleFunc(pattern string, handler http.HandlerFunc) {
	h.mux.HandleFunc(pattern, handler)
}

// setupRoutes configures HTTP routes.
func (h *HTTPEndpoint) setupRoutes() {
	h.mux.HandleFunc("/", h.handleRoot)
	h.mux.HandleFunc("/api/", h.handleAPI)
	h.mux.HandleFunc("/ws/", h.handleWebSocket)
	// Note: /SESSION-ID/variables is handled in handleRoot
}

// ServeHTTP implements http.Handler.
func (h *HTTPEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleRoot handles root and session paths.
func (h *HTTPEndpoint) handleRoot(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root path handling
	if path == "/" {
		// Check for custom root session provider (e.g., ui-mcp)
		if h.rootSessionProvider != nil {
			if sessionID := h.rootSessionProvider(); sessionID != "" {
				// Serve index.html with session cookie (no redirect)
				h.setSessionCookie(w, sessionID)
				h.serveStatic(w, r, "index.html")
				return
			}
		}
		// Default: create new session and redirect
		sess, _, err := h.sessions.CreateSession()
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		// Use internal session ID for URL path (user-facing)
		http.Redirect(w, r, "/"+sess.ID, http.StatusTemporaryRedirect)
		return
	}

	// Extract session ID from path (e.g., /abc123 or /abc123/page)
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	sessionID := parts[0]

	// Check if this is a valid session
	if h.sessions.SessionExists(sessionID) {
		// Set session cookie for this session
		h.setSessionCookie(w, sessionID)
		// CRC: crc-HTTPEndpoint.md (R57, R58)
		if len(parts) > 1 {
			switch parts[1] {
			case "variables":
				h.ServeVariableBrowser(w, r)
				return
			case "variables.json":
				h.HandleVariablesJSON(w, r, sessionID)
				return
			}
		}
		// Serve the SPA - it will handle the routing client-side
		h.serveStatic(w, r, "index.html")
		return
	}

	// Not a session path - serve static file
	h.serveStatic(w, r, strings.TrimPrefix(path, "/"))
}

// setSessionCookie sets the ui-session cookie.
func (h *HTTPEndpoint) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "ui-session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: false, // JS needs to read it
		SameSite: http.SameSiteLaxMode,
	})
}

// serveStatic serves a static file.
func (h *HTTPEndpoint) serveStatic(w http.ResponseWriter, r *http.Request, path string) {
	if path == "" {
		path = "index.html"
	}

	// Set content type based on extension (http.ServeFile uses content sniffing which fails for CSS)
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	// Try custom directory first
	if h.staticDir != "" {
		http.ServeFile(w, r, h.staticDir+"/"+path)
		return
	}

	// Fall back to embedded site
	if h.embeddedSite != nil {
		data, err := fs.ReadFile(h.embeddedSite, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Write(data)
		return
	}

	http.NotFound(w, r)
}

// handleWebSocket handles WebSocket upgrade requests.
func (h *HTTPEndpoint) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path: /ws/SESSION-ID
	path := strings.TrimPrefix(r.URL.Path, "/ws/")
	sessionID := strings.Split(path, "/")[0]

	if !h.sessions.SessionExists(sessionID) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	h.wsEndpoint.HandleWebSocket(w, r, sessionID)
}

// handleAPI handles REST API requests.
func (h *HTTPEndpoint) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract endpoint: /api/create, /api/update, etc.
	endpoint := strings.TrimPrefix(r.URL.Path, "/api/")

	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		h.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Override type from URL path
	msg.Type = protocol.MessageType(endpoint)

	// Use a synthetic connection ID for API calls
	connectionID := "api-" + r.RemoteAddr

	resp, err := h.handler.HandleMessage(connectionID, &msg)
	if err != nil {
		h.writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// writeError writes an error response.
func (h *HTTPEndpoint) writeError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(protocol.Response{Error: message})
}

// HandleProtocolCommand processes CLI protocol commands.
func (h *HTTPEndpoint) HandleProtocolCommand(msg *protocol.Message) (*protocol.Response, error) {
	return h.handler.HandleMessage("cli", msg)
}

// ServeVariableBrowser serves the embedded variable browser HTML page.
// CRC: crc-HTTPEndpoint.md (R58)
func (h *HTTPEndpoint) ServeVariableBrowser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(variableBrowserHTML))
}

// HandleVariablesJSON returns variable data as JSON.
// CRC: crc-HTTPEndpoint.md (R57, R62, R80, R81)
func (h *HTTPEndpoint) HandleVariablesJSON(w http.ResponseWriter, r *http.Request, sessionID string) {
	vendedID := h.sessions.GetVendedID(sessionID)
	if vendedID == "" {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if h.debugDataProvider == nil {
		json.NewEncoder(w).Encode([]DebugVariable{})
		return
	}

	diagLevel := 0
	if d := r.URL.Query().Get("diag"); d != "" {
		fmt.Sscanf(d, "%d", &diagLevel)
	}

	variables, changeCount, err := h.debugDataProvider(vendedID, diagLevel)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("X-Change-Count", strconv.FormatInt(changeCount, 10))
	json.NewEncoder(w).Encode(variables)
}
