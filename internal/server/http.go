// CRC: crc-HTTPEndpoint.md
// Spec: interfaces.md, deployment.md
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/zot/ui-engine/internal/protocol"
	"github.com/zot/ui-engine/internal/session"
)

// HTTPEndpoint handles HTTP requests.
type HTTPEndpoint struct {
	sessions    *session.Manager
	handler     *protocol.Handler
	wsEndpoint  *WebSocketEndpoint
	staticDir    string
	embeddedSite fs.FS
	mux          *http.ServeMux
}

// NewHTTPEndpoint creates a new HTTP endpoint.
func NewHTTPEndpoint(sessions *session.Manager, handler *protocol.Handler, wsEndpoint *WebSocketEndpoint) *HTTPEndpoint {
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

// setupRoutes configures HTTP routes.
func (h *HTTPEndpoint) setupRoutes() {
	h.mux.HandleFunc("/", h.handleRoot)
	h.mux.HandleFunc("/api/", h.handleAPI)
	h.mux.HandleFunc("/ws/", h.handleWebSocket)
}

// ServeHTTP implements http.Handler.
func (h *HTTPEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleRoot handles root and session paths.
func (h *HTTPEndpoint) handleRoot(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root path - create new session and redirect
	if path == "/" {
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
		// Serve the SPA - it will handle the routing client-side
		h.serveStatic(w, r, "index.html")
		return
	}

	// Not a session path - serve static file
	h.serveStatic(w, r, strings.TrimPrefix(path, "/"))
}

// serveStatic serves a static file.
func (h *HTTPEndpoint) serveStatic(w http.ResponseWriter, r *http.Request, path string) {
	if path == "" {
		path = "index.html"
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

		// Set content type based on extension
		if strings.HasSuffix(path, ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".json") {
			w.Header().Set("Content-Type", "application/json")
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
