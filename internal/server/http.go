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
	"strings"

	"github.com/zot/ui-engine/internal/protocol"
)

// DebugDataProvider is called to get variable data for the debug page.
type DebugDataProvider func(sessionID string) ([]DebugVariable, error)

// RootSessionProvider returns the session ID to use for the root path "/".
// If it returns an empty string, the default behavior (create new session and redirect) is used.
// If it returns a session ID, index.html is served with a session cookie set.
type RootSessionProvider func() string

// DebugVariable represents a variable for the debug tree view.
type DebugVariable struct {
	ID         int64             `json:"id"`
	ParentID   int64             `json:"parentId"`
	Type       string            `json:"type,omitempty"`
	Path       string            `json:"path,omitempty"`
	Value      any               `json:"value,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	ChildIDs   []int64           `json:"childIds,omitempty"`
	Error      string            `json:"error,omitempty"`
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
		// Check for /SESSION-ID/variables debug endpoint
		if len(parts) > 1 && parts[1] == "variables" {
			h.handleDebugVariables(w, r, sessionID)
			return
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

// handleDebugVariables renders a debug page with a variable tree.
// sessionID is the internal UUID from the URL path.
func (h *HTTPEndpoint) handleDebugVariables(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Translate internal session ID (UUID) to vended ID (numeric)
	vendedID := h.sessions.GetVendedID(sessionID)
	if vendedID == "" {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get variable data using vended ID
	var variables []DebugVariable
	var dataErr error
	if h.debugDataProvider != nil {
		variables, dataErr = h.debugDataProvider(vendedID)
	}

	// Generate HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `<!DOCTYPE html>
<html>
<head>
  <title>Debug: Variables</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.19.1/cdn/themes/light.css" />
  <script type="module" src="https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.19.1/cdn/shoelace-autoloader.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; padding: 20px; max-width: 1200px; margin: 0 auto; }
    h1 { color: #333; }
    .error { color: red; padding: 10px; background: #fee; border-radius: 4px; }
    .tree-container { margin-top: 20px; }
    sl-tree { --indent-size: 20px; }
    sl-tree-item::part(item) { padding: 4px 0; }
    .var-id { color: #666; font-size: 0.9em; margin-right: 8px; }
    .var-type { color: #0066cc; font-weight: bold; margin-right: 8px; }
    .var-path { color: #666; font-style: italic; margin-right: 8px; }
    .var-value { color: #228b22; font-family: monospace; font-size: 0.9em; }
    .var-error { color: #cc0000; font-family: monospace; font-size: 0.9em; background: #fee; padding: 2px 6px; border-radius: 3px; margin-left: 8px; }
    .var-props { color: #888; font-size: 0.8em; margin-left: 16px; }
    .refresh-btn { margin-bottom: 16px; }
    pre { background: #f5f5f5; padding: 10px; border-radius: 4px; overflow-x: auto; }
  </style>
</head>
<body>
  <h1>Variable Tree - Session ` + sessionID + `</h1>
  <sl-button class="refresh-btn" onclick="location.reload()">
    <sl-icon slot="prefix" name="arrow-clockwise"></sl-icon>
    Refresh
  </sl-button>
`

	if dataErr != nil {
		html += `<div class="error">Error: ` + dataErr.Error() + `</div>`
	} else if len(variables) == 0 {
		html += `<div class="error">No variables found for session ` + sessionID + `</div>`
	} else {
		// Build tree HTML
		html += `<div class="tree-container"><sl-tree>`
		html += h.renderVariableTree(variables)
		html += `</sl-tree></div>`

		// Also show raw JSON
		jsonBytes, _ := json.MarshalIndent(variables, "", "  ")
		html += `<h2>Raw JSON</h2><pre>` + string(jsonBytes) + `</pre>`
	}

	html += `</body></html>`
	w.Write([]byte(html))
}

// renderVariableTree renders variables as nested sl-tree-item elements.
func (h *HTTPEndpoint) renderVariableTree(variables []DebugVariable) string {
	// Build a map for quick lookup
	varMap := make(map[int64]DebugVariable)
	for _, v := range variables {
		varMap[v.ID] = v
	}

	// Find roots (parentID == 0)
	var roots []int64
	for _, v := range variables {
		if v.ParentID == 0 {
			roots = append(roots, v.ID)
		}
	}

	// Render recursively
	var result strings.Builder
	for _, rootID := range roots {
		h.renderVariableNode(&result, varMap, rootID)
	}
	return result.String()
}

// renderVariableNode renders a single variable and its children.
func (h *HTTPEndpoint) renderVariableNode(sb *strings.Builder, varMap map[int64]DebugVariable, varID int64) {
	v, ok := varMap[varID]
	if !ok {
		return
	}

	// Format value as JSON
	valueStr := ""
	if v.Value != nil {
		valueBytes, _ := json.Marshal(v.Value)
		valueStr = string(valueBytes)
		if len(valueStr) > 100 {
			valueStr = valueStr[:100] + "..."
		}
	}

	// Build label
	label := `<span class="var-id">#` + fmt.Sprintf("%d", v.ID) + `</span>`
	if v.Type != "" {
		label += `<span class="var-type">` + v.Type + `</span>`
	}
	if v.Path != "" {
		label += `<span class="var-path">` + v.Path + `</span>`
	}
	if valueStr != "" {
		label += `<span class="var-value">` + escapeHTML(valueStr) + `</span>`
	}
	// CRC: crc-HTTPEndpoint.md (R23, R24, R25)
	if v.Error != "" {
		label += `<span class="var-error">` + escapeHTML(v.Error) + `</span>`
	}

	hasChildren := len(v.ChildIDs) > 0

	if hasChildren {
		sb.WriteString(`<sl-tree-item expanded>`)
	} else {
		sb.WriteString(`<sl-tree-item>`)
	}
	sb.WriteString(label)

	// Render properties if any (excluding type and path which are already shown)
	if len(v.Properties) > 0 {
		sb.WriteString(`<div class="var-props">`)
		first := true
		for k, val := range v.Properties {
			if k == "type" || k == "path" {
				continue
			}
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(k + "=" + val)
			first = false
		}
		sb.WriteString(`</div>`)
	}

	// Render children
	for _, childID := range v.ChildIDs {
		h.renderVariableNode(sb, varMap, childID)
	}

	sb.WriteString(`</sl-tree-item>`)
}

// escapeHTML escapes special HTML characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
