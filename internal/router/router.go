// CRC: crc-Router.md
// Spec: interfaces.md
package router

import (
	"strings"
	"sync"
)

// Route maps a URL path to a presenter variable ID.
type Route struct {
	Path       string // URL path pattern (without session prefix)
	VariableID int64  // Associated presenter variable ID
}

// Router handles URL routing for a session.
type Router struct {
	sessionID string
	routes    map[string]*Route
	mu        sync.RWMutex
}

// New creates a new router for the given session.
func New(sessionID string) *Router {
	return &Router{
		sessionID: sessionID,
		routes:    make(map[string]*Route),
	}
}

// Register associates a URL path with a presenter variable.
func (r *Router) Register(path string, variableID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize path (ensure leading slash, no trailing slash)
	path = normalizePath(path)

	r.routes[path] = &Route{
		Path:       path,
		VariableID: variableID,
	}
}

// Unregister removes a URL path mapping.
func (r *Router) Unregister(path string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	path = normalizePath(path)
	if _, ok := r.routes[path]; ok {
		delete(r.routes, path)
		return true
	}
	return false
}

// Resolve finds the presenter variable ID for a URL path.
// Returns the variable ID and true if found, 0 and false otherwise.
func (r *Router) Resolve(path string) (int64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	path = normalizePath(path)
	if route, ok := r.routes[path]; ok {
		return route.VariableID, true
	}
	return 0, false
}

// Match checks if a URL matches any registered pattern.
func (r *Router) Match(path string) bool {
	_, found := r.Resolve(path)
	return found
}

// BuildURL constructs the full URL for a presenter path.
// Returns /{sessionID}/{path} format.
func (r *Router) BuildURL(path string) string {
	path = normalizePath(path)
	if path == "/" {
		return "/" + r.sessionID
	}
	return "/" + r.sessionID + path
}

// ParseURL extracts session ID and path from a full URL path.
// Input: /SESSION-ID/some/path
// Returns: sessionID, path (e.g., "SESSION-ID", "/some/path")
func ParseURL(urlPath string) (sessionID, path string) {
	// Remove leading slash
	urlPath = strings.TrimPrefix(urlPath, "/")

	// Split on first slash
	parts := strings.SplitN(urlPath, "/", 2)
	if len(parts) == 0 {
		return "", "/"
	}

	sessionID = parts[0]
	if len(parts) == 1 {
		path = "/"
	} else {
		path = "/" + parts[1]
	}
	return sessionID, path
}

// IsRegisteredPath checks if a path was explicitly registered.
func (r *Router) IsRegisteredPath(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	path = normalizePath(path)
	_, ok := r.routes[path]
	return ok
}

// GetSessionID returns the session ID for this router.
func (r *Router) GetSessionID() string {
	return r.sessionID
}

// GetRoutes returns a copy of all registered routes.
func (r *Router) GetRoutes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]Route, 0, len(r.routes))
	for _, route := range r.routes {
		routes = append(routes, *route)
	}
	return routes
}

// normalizePath ensures path has leading slash and no trailing slash.
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}
