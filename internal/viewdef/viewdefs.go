// ViewdefManager loads and serves viewdefs to frontend sessions.
// Spec: viewdefs.md
package viewdef

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zot/ui/internal/bundle"
)

// ViewdefManager manages viewdef loading and tracking.
type ViewdefManager struct {
	// viewdefs maps TYPE.NAMESPACE to HTML content
	viewdefs map[string]string
	// sentTypes tracks which types have been sent per session
	// sessionID -> set of type names
	sentTypes map[string]map[string]bool
	mu        sync.RWMutex
}

// NewViewdefManager creates a new viewdef manager.
func NewViewdefManager() *ViewdefManager {
	return &ViewdefManager{
		viewdefs:  make(map[string]string),
		sentTypes: make(map[string]map[string]bool),
	}
}

// LoadFromDirectory loads viewdefs from a directory.
// Files should be named TYPE.NAMESPACE.html (e.g., Adder.DEFAULT.html)
func (m *ViewdefManager) LoadFromDirectory(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Get filename without directory
		filename := filepath.Base(path)
		// Remove .html extension to get TYPE.NAMESPACE
		key := strings.TrimSuffix(filename, ".html")

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		m.viewdefs[key] = string(content)
		return nil
	})
}

// LoadFromBundle loads viewdefs from the embedded bundle.
func (m *ViewdefManager) LoadFromBundle() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := bundle.ListFilesInDir("viewdefs")
	if err != nil {
		return err
	}

	for _, filePath := range files {
		if !strings.HasSuffix(filePath, ".html") {
			continue
		}

		content, err := bundle.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Get filename from path (e.g., "viewdefs/Adder.DEFAULT.html" -> "Adder.DEFAULT")
		filename := filepath.Base(filePath)
		key := strings.TrimSuffix(filename, ".html")

		m.viewdefs[key] = string(content)
	}

	return nil
}

// LoadFromFS loads viewdefs from a filesystem (for custom site directories).
func (m *ViewdefManager) LoadFromFS(fsys fs.FS, dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		filename := filepath.Base(path)
		key := strings.TrimSuffix(filename, ".html")

		m.viewdefs[key] = string(content)
		return nil
	})
}

// GetViewdefsForType returns all viewdefs for a given type.
// Returns a map of TYPE.NAMESPACE -> HTML content
func (m *ViewdefManager) GetViewdefsForType(typeName string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	prefix := typeName + "."

	for key, content := range m.viewdefs {
		if strings.HasPrefix(key, prefix) {
			result[key] = content
		}
	}

	return result
}

func (m *ViewdefManager) GetSent(sessionID string) map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentTypes[sessionID]
}

// GetNewViewdefsForSession returns viewdefs for a type that haven't been sent to this session yet.
// Marks the type as sent for this session.
func (m *ViewdefManager) GetNewViewdefsForSession(sessionID, typeName string) map[string]string {
	defs := make(map[string]string)
	m.AddNewViewdefsForSession(sessionID, typeName, defs)
	return defs
}

// GetNewViewdefsForSession returns viewdefs for a type that haven't been sent to this session yet.
// Marks the type as sent for this session.
func (m *ViewdefManager) AddNewViewdefsForSession(sessionID, typeName string, defs map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize session tracking if needed
	if m.sentTypes[sessionID] == nil {
		m.sentTypes[sessionID] = make(map[string]bool)
	}

	// Check if already sent
	if m.sentTypes[sessionID][typeName] {
		return
	}

	// Mark as sent
	m.sentTypes[sessionID][typeName] = true

	// Get viewdefs for this type
	prefix := typeName + "."

	for key, content := range m.viewdefs {
		if strings.HasPrefix(key, prefix) {
			defs[key] = content
		}
	}
}

// ClearSession removes tracking data for a session.
func (m *ViewdefManager) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sentTypes, sessionID)
}

// GetAllViewdefs returns all loaded viewdefs.
func (m *ViewdefManager) GetAllViewdefs() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.viewdefs))
	for k, v := range m.viewdefs {
		result[k] = v
	}
	return result
}

// Count returns the number of loaded viewdefs.
func (m *ViewdefManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.viewdefs)
}
