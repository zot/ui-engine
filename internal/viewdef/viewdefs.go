// ViewdefManager loads and serves viewdefs to frontend sessions.
// CRC: crc-ViewdefManager.md | Seq: seq-app-discovery.md
// Spec: viewdefs.md, app-discovery.md
package viewdef

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zot/ui-engine/internal/bundle"
	"github.com/zot/ui-engine/internal/site"
)

// viewdefEntry tracks a viewdef's content and source file info.
type viewdefEntry struct {
	content  string
	filePath string    // Source file path (empty if from bundle or dynamic)
	modTime  time.Time // Last modification time when loaded
	rank     int       // Index of the owning source; lower outranks higher
}

// ViewdefManager manages viewdef loading and tracking.
type ViewdefManager struct {
	// viewdefs maps TYPE.NAMESPACE to viewdef entry
	viewdefs map[string]*viewdefEntry
	// sentViewdefs tracks which viewdefs have been sent per session with their modTime
	// sessionID -> viewdef key -> modTime when sent
	sentViewdefs map[string]map[string]time.Time
	// viewdefDir is where AddViewdef writes dynamic viewdefs
	viewdefDir string
	// sources are the ranked viewdef directories (see site.ViewdefSources)
	sources []site.Source
	// collisions records key|loserDir pairs already logged
	collisions map[string]bool
	logf       func(level int, format string, args ...interface{})
	mu         sync.RWMutex
}

// NewViewdefManager creates a new viewdef manager.
func NewViewdefManager() *ViewdefManager {
	return &ViewdefManager{
		viewdefs:     make(map[string]*viewdefEntry),
		sentViewdefs: make(map[string]map[string]time.Time),
		collisions:   make(map[string]bool),
		logf: func(level int, format string, args ...interface{}) {
			log.Printf(format, args...)
		},
	}
}

// SetLogger routes collision and load messages through the host's logger.
func (m *ViewdefManager) SetLogger(logf func(level int, format string, args ...interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logf = logf
}

// SetViewdefDir sets the directory for on-demand viewdef loading.
func (m *ViewdefManager) SetViewdefDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viewdefDir = dir
}

// noRank marks entries that no source owns (bundle or dynamic).
const noRank = -1

// LoadFromSources registers viewdefs from ranked sources, discarding any
// previously file-sourced entries. The first source to offer a key owns it;
// a later offer is a collision, logged and skipped.
// CRC: crc-ViewdefManager.md | Seq: seq-app-discovery.md#1.4 | R125, R126, R127
func (m *ViewdefManager) LoadFromSources(sources []site.Source) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadFromSourcesLocked(sources)
}

// SetSources replaces the source list and reloads, returning the keys whose
// content changed (new, replaced or newly owned) so callers can re-push them.
// CRC: crc-ViewdefManager.md | Seq: seq-app-discovery.md#2.1.1 | R129
func (m *ViewdefManager) SetSources(sources []site.Source) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := make(map[string]string, len(m.viewdefs))
	for key, entry := range m.viewdefs {
		old[key] = entry.content
	}
	m.loadFromSourcesLocked(sources)
	changed := make(map[string]string)
	for key, entry := range m.viewdefs {
		if prev, ok := old[key]; !ok || prev != entry.content {
			changed[key] = entry.content
		}
	}
	return changed
}

func (m *ViewdefManager) loadFromSourcesLocked(sources []site.Source) {
	m.sources = sources
	for key, entry := range m.viewdefs {
		if entry.filePath != "" {
			delete(m.viewdefs, key)
		}
	}
	for rank, src := range sources {
		filepath.Walk(src.Dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// One bad entry (e.g. a broken symlink) must not abort the walk.
				m.logf(1, "viewdef load: skipping %s: %v", path, err)
				return nil
			}
			if info.IsDir() || !strings.HasSuffix(path, ".html") {
				return nil
			}
			m.offerLocked(rank, path, info.ModTime())
			return nil
		})
	}
}

// offerLocked registers path for its key unless a higher-ranked source owns
// it; returns false for a collision. Must be called with the write lock held.
// CRC: crc-ViewdefManager.md | R126, R127
func (m *ViewdefManager) offerLocked(rank int, path string, modTime time.Time) bool {
	key := strings.TrimSuffix(filepath.Base(path), ".html")
	if existing, ok := m.viewdefs[key]; ok && existing.rank != noRank && existing.rank < rank {
		if existing.filePath != path {
			m.logCollisionLocked(key, existing.rank, rank)
		}
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		m.logf(1, "viewdef load: skipping %s: %v", path, err)
		return false
	}
	m.viewdefs[key] = &viewdefEntry{content: string(content), filePath: path, modTime: modTime, rank: rank}
	return true
}

// logCollisionLocked logs a collision once per key and losing source.
// CRC: crc-ViewdefManager.md | R127
func (m *ViewdefManager) logCollisionLocked(key string, winner, loser int) {
	id := key + "|" + m.sources[loser].Dir
	if m.collisions[id] {
		return
	}
	m.collisions[id] = true
	m.logf(0, "viewdef collision: %s from %s wins over %s", key, m.sources[winner].Label, m.sources[loser].Label)
}

// ApplyFileChange reloads one changed viewdef file, returning its key and
// content. ok is false when the file belongs to no source, cannot be read, or
// its key is owned by a higher-ranked source.
// CRC: crc-ViewdefManager.md | Seq: seq-viewdef-hotload.md | R126
func (m *ViewdefManager) ApplyFileChange(path string) (key, content string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rank := m.rankOfLocked(path)
	if rank == noRank {
		return "", "", false
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", "", false
	}
	if !m.offerLocked(rank, path, info.ModTime()) {
		return "", "", false
	}
	key = strings.TrimSuffix(filepath.Base(path), ".html")
	return key, m.viewdefs[key].content, true
}

// rankOfLocked returns the index of the source containing path, or noRank.
func (m *ViewdefManager) rankOfLocked(path string) int {
	for rank, src := range m.sources {
		if strings.HasPrefix(path, src.Dir+string(filepath.Separator)) {
			return rank
		}
	}
	return noRank
}

// AddViewdef adds or updates a viewdef dynamically.
// If viewdefDir is set, writes to file and tracks the file path.
func (m *ViewdefManager) AddViewdef(key, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &viewdefEntry{content: content, rank: noRank}

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
		m.viewdefs[key] = &viewdefEntry{content: string(content), rank: noRank}
	}

	return nil
}

// LoadFromFS loads viewdefs from a filesystem (for custom site directories).
func (m *ViewdefManager) LoadFromFS(fsys fs.FS, dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("viewdef load: skipping %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			log.Printf("viewdef load: skipping %s: %v", path, err)
			return nil
		}

		filename := filepath.Base(path)
		key := strings.TrimSuffix(filename, ".html")

		// FS viewdefs have no trackable file path
		m.viewdefs[key] = &viewdefEntry{content: string(content), rank: noRank}
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

// tryLoadFromFilesystem loads a type's viewdefs on demand, searching the
// sources in rank order under the same first-wins rule as LoadFromSources.
// Must be called with lock held.
// CRC: crc-ViewdefManager.md | R128
func (m *ViewdefManager) tryLoadFromFilesystem(typeName string) {
	for rank, src := range m.sources {
		matches, err := filepath.Glob(filepath.Join(src.Dir, typeName+".*.html"))
		if err != nil {
			continue
		}
		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			key := strings.TrimSuffix(filepath.Base(path), ".html")
			// Skip a file already loaded with the same or newer mod time.
			if existing, ok := m.viewdefs[key]; ok && existing.filePath == path && !info.ModTime().After(existing.modTime) {
				continue
			}
			m.offerLocked(rank, path, info.ModTime())
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

// Sources returns the current ranked viewdef sources.
func (m *ViewdefManager) Sources() []site.Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]site.Source(nil), m.sources...)
}
