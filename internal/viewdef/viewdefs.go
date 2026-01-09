// ViewdefManager loads and serves viewdefs to frontend sessions.
// Spec: viewdefs.md
package viewdef

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zot/ui-engine/internal/bundle"
)

// viewdefEntry tracks a viewdef's content and source file info.
type viewdefEntry struct {
	content  string
	filePath string    // Source file path (empty if from bundle or dynamic)
	modTime  time.Time // Last modification time when loaded
}

// ViewdefManager manages viewdef loading and tracking.
type ViewdefManager struct {
	// viewdefs maps TYPE.NAMESPACE to viewdef entry
	viewdefs map[string]*viewdefEntry
	// sentViewdefs tracks which viewdefs have been sent per session with their modTime
	// sessionID -> viewdef key -> modTime when sent
	sentViewdefs map[string]map[string]time.Time
	// viewdefDir is the directory to check for viewdefs on-demand
	viewdefDir string
	mu         sync.RWMutex
}

// NewViewdefManager creates a new viewdef manager.
func NewViewdefManager() *ViewdefManager {
	return &ViewdefManager{
		viewdefs:     make(map[string]*viewdefEntry),
		sentViewdefs: make(map[string]map[string]time.Time),
	}
}

// SetViewdefDir sets the directory for on-demand viewdef loading.
func (m *ViewdefManager) SetViewdefDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viewdefDir = dir
}

// LoadFromDirectory loads viewdefs from a directory.
// Files should be named TYPE.NAMESPACE.html (e.g., Adder.DEFAULT.html)
func (m *ViewdefManager) LoadFromDirectory(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store directory for on-demand loading
	m.viewdefDir = dir

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

		m.viewdefs[key] = &viewdefEntry{
			content:  string(content),
			filePath: path,
			modTime:  info.ModTime(),
		}
		return nil
	})
}

// AddViewdef adds or updates a viewdef dynamically.
// If viewdefDir is set, writes to file and tracks the file path.
func (m *ViewdefManager) AddViewdef(key, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &viewdefEntry{content: content}

	// If we have a viewdef directory, write to file and track it
	if m.viewdefDir != "" {
		filePath := filepath.Join(m.viewdefDir, key+".html")
		if err := os.WriteFile(filePath, []byte(content), 0644); err == nil {
			if info, err := os.Stat(filePath); err == nil {
				entry.filePath = filePath
				entry.modTime = info.ModTime()
			}
		}
	}

	m.viewdefs[key] = entry
}

// LoadFromBundle loads viewdefs from the embedded bundle.
func (m *ViewdefManager) LoadFromBundle() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := bundle.ListFilesInDir("viewdefs")
	if err != nil {
		return err
	}

	for _, bundlePath := range files {
		if !strings.HasSuffix(bundlePath, ".html") {
			continue
		}

		content, err := bundle.ReadFile(bundlePath)
		if err != nil {
			continue
		}

		// Get filename from path (e.g., "viewdefs/Adder.DEFAULT.html" -> "Adder.DEFAULT")
		filename := filepath.Base(bundlePath)
		key := strings.TrimSuffix(filename, ".html")

		// Bundle viewdefs have no file path (embedded)
		m.viewdefs[key] = &viewdefEntry{content: string(content)}
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

		// FS viewdefs have no trackable file path
		m.viewdefs[key] = &viewdefEntry{content: string(content)}
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

	for key, entry := range m.viewdefs {
		if strings.HasPrefix(key, prefix) {
			result[key] = entry.content
		}
	}

	return result
}

// reloadIfStale checks if a viewdef's file has been modified and reloads if needed.
// Must be called with write lock held.
func (m *ViewdefManager) reloadIfStale(key string, entry *viewdefEntry) {
	if entry.filePath == "" {
		return
	}

	info, err := os.Stat(entry.filePath)
	if err != nil {
		return
	}

	if info.ModTime().After(entry.modTime) {
		content, err := os.ReadFile(entry.filePath)
		if err != nil {
			return
		}
		entry.content = string(content)
		entry.modTime = info.ModTime()
	}
}

// LoadViewdefsForType loads viewdefs for a type from filesystem into cache.
// Does not mark them as sent - use GetChangedViewdefsForSession after to get them.
func (m *ViewdefManager) LoadViewdefsForType(typeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tryLoadFromFilesystem(typeName)
}

// tryLoadFromFilesystem attempts to load viewdefs for a type from the filesystem.
// Must be called with lock held.
func (m *ViewdefManager) tryLoadFromFilesystem(typeName string) {
	if m.viewdefDir == "" {
		return
	}

	// Look for TYPE.*.html files
	pattern := filepath.Join(m.viewdefDir, typeName+".*.html")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, path := range matches {
		filename := filepath.Base(path)
		key := strings.TrimSuffix(filename, ".html")

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		// Skip if already loaded with same or newer mod time
		if existing, exists := m.viewdefs[key]; exists {
			if existing.filePath != "" && !info.ModTime().After(existing.modTime) {
				continue
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		m.viewdefs[key] = &viewdefEntry{
			content:  string(content),
			filePath: path,
			modTime:  info.ModTime(),
		}
	}
}

// GetChangedViewdefsForSession returns viewdefs that need to be sent to a session.
// This includes:
// - Viewdefs for new types that haven't been sent yet
// - Viewdefs that have been modified since they were last sent
// Marks returned viewdefs as sent with their current mod time.
func (m *ViewdefManager) GetChangedViewdefsForSession(sessionID string) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize session tracking if needed
	if m.sentViewdefs[sessionID] == nil {
		m.sentViewdefs[sessionID] = make(map[string]time.Time)
	}

	defs := make(map[string]string)
	sentTimes := m.sentViewdefs[sessionID]

	for key, entry := range m.viewdefs {
		// Check for file changes
		m.reloadIfStale(key, entry)

		sentTime, wasSent := sentTimes[key]
		if !wasSent || entry.modTime.After(sentTime) {
			defs[key] = entry.content
			sentTimes[key] = entry.modTime
		}
	}

	return defs
}

// AddNewViewdefsForType loads and marks viewdefs for a type as sent.
// This is called when a new type is encountered in the variable changes.
func (m *ViewdefManager) AddNewViewdefsForType(sessionID, typeName string, defs map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize session tracking if needed
	if m.sentViewdefs[sessionID] == nil {
		m.sentViewdefs[sessionID] = make(map[string]time.Time)
	}

	// Try to load from filesystem if not in cache
	m.tryLoadFromFilesystem(typeName)

	// Get viewdefs for this type
	prefix := typeName + "."
	sentTimes := m.sentViewdefs[sessionID]

	for key, entry := range m.viewdefs {
		if strings.HasPrefix(key, prefix) {
			// Check for file changes
			m.reloadIfStale(key, entry)

			sentTime, wasSent := sentTimes[key]
			if !wasSent || entry.modTime.After(sentTime) {
				defs[key] = entry.content
				sentTimes[key] = entry.modTime
			}
		}
	}
}

// ClearSession removes tracking data for a session.
func (m *ViewdefManager) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sentViewdefs, sessionID)
}

// GetAllViewdefs returns all loaded viewdefs.
func (m *ViewdefManager) GetAllViewdefs() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.viewdefs))
	for k, entry := range m.viewdefs {
		result[k] = entry.content
	}
	return result
}

// Count returns the number of loaded viewdefs.
func (m *ViewdefManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.viewdefs)
}

// updateViewdef updates a viewdef entry (used by HotLoader).
// This bypasses the normal loading mechanism to update in-place.
func (m *ViewdefManager) updateViewdef(key, content, filePath string, modTime time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.viewdefs[key] = &viewdefEntry{
		content:  content,
		filePath: filePath,
		modTime:  modTime,
	}
}

// hasSessionReceivedViewdef checks if a session has received a specific viewdef.
func (m *ViewdefManager) hasSessionReceivedViewdef(sessionID, key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sentTimes, ok := m.sentViewdefs[sessionID]
	if !ok {
		return false
	}
	_, received := sentTimes[key]
	return received
}

// GetSessionsForViewdef returns all session IDs that have received a viewdef.
func (m *ViewdefManager) GetSessionsForViewdef(key string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []string
	for sessionID, sentTimes := range m.sentViewdefs {
		if _, ok := sentTimes[key]; ok {
			sessions = append(sessions, sessionID)
		}
	}
	return sessions
}

// MarkViewdefSent marks a viewdef as sent for a session.
// Used when pushing viewdefs outside of normal flow (e.g., hot-reload).
func (m *ViewdefManager) MarkViewdefSent(sessionID, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sentViewdefs[sessionID] == nil {
		m.sentViewdefs[sessionID] = make(map[string]time.Time)
	}

	entry, ok := m.viewdefs[key]
	if ok {
		m.sentViewdefs[sessionID][key] = entry.modTime
	}
}
