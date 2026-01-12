// Package viewdef provides viewdef hot-loading support.
// CRC: crc-ViewdefStore.md
// Spec: viewdefs.md
// Sequence: seq-viewdef-hotload.md
package viewdef

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zot/ui-engine/internal/config"
)

// SessionPusher provides session management for hot-loading viewdefs.
// CRC: crc-ViewdefStore.md
type SessionPusher interface {
	// GetSessionIDs returns a list of active vended session IDs.
	GetSessionIDs() []string
	// PushViewdefs pushes updated viewdefs to a session.
	PushViewdefs(sessionID string, viewdefs map[string]string)
}

// HotLoader watches the viewdef directory for file changes and triggers pushes.
type HotLoader struct {
	config     *config.Config
	viewdefDir string
	watcher    *fsnotify.Watcher
	manager    *ViewdefManager
	sessions   SessionPusher

	// Symlink tracking (see cross-cutting: Hot-Loading Symlink Tracking)
	symlinkTargets map[string]string // viewdef file path -> resolved target dir
	watchedDirs    map[string]int    // dir path -> reference count
	mu             sync.Mutex

	// Debouncing
	pendingReloads map[string]time.Time
	debounceMu     sync.Mutex
	debounceDelay  time.Duration

	done chan struct{}
}

// NewHotLoader creates a new hot loader for the given viewdef directory.
func NewHotLoader(cfg *config.Config, viewdefDir string, manager *ViewdefManager, sessions SessionPusher) (*HotLoader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	h := &HotLoader{
		config:         cfg,
		viewdefDir:     viewdefDir,
		watcher:        watcher,
		manager:        manager,
		sessions:       sessions,
		symlinkTargets: make(map[string]string),
		watchedDirs:    make(map[string]int),
		pendingReloads: make(map[string]time.Time),
		debounceDelay:  100 * time.Millisecond,
		done:           make(chan struct{}),
	}

	return h, nil
}

// Start begins watching for file changes.
func (h *HotLoader) Start() error {
	// Watch the main viewdef directory
	if err := h.addWatch(h.viewdefDir); err != nil {
		return err
	}

	// Scan for existing symlinks and watch their target directories
	if err := h.scanSymlinks(); err != nil {
		h.config.Log(1, "ViewdefHotLoader: error scanning symlinks: %v", err)
	}

	// Start the event loop
	go h.eventLoop()

	// Start the debounce processor
	go h.debounceLoop()

	h.config.Log(1, "ViewdefHotLoader: watching %s for changes", h.viewdefDir)
	return nil
}

// Stop stops the hot loader.
func (h *HotLoader) Stop() error {
	close(h.done)
	return h.watcher.Close()
}

// eventLoop processes file system events.
func (h *HotLoader) eventLoop() {
	for {
		select {
		case <-h.done:
			return
		case event, ok := <-h.watcher.Events:
			if !ok {
				return
			}
			h.handleEvent(event)
		case err, ok := <-h.watcher.Errors:
			if !ok {
				return
			}
			h.config.Log(1, "ViewdefHotLoader: watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event.
func (h *HotLoader) handleEvent(event fsnotify.Event) {
	// Only care about .html files
	if !strings.HasSuffix(event.Name, ".html") {
		return
	}

	h.config.Log(3, "ViewdefHotLoader: event %s on %s", event.Op, event.Name)

	// Handle symlink changes in the viewdef directory
	if filepath.Dir(event.Name) == h.viewdefDir {
		switch {
		case event.Op&fsnotify.Create != 0:
			h.updateSymlinkWatch(event.Name)
		case event.Op&fsnotify.Remove != 0:
			h.removeSymlinkWatch(event.Name)
		case event.Op&fsnotify.Rename != 0:
			// Rename is like remove + create elsewhere
			h.removeSymlinkWatch(event.Name)
		}
	}

	// Queue reload for write events
	if event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0 {
		h.queueReload(event.Name)
	}
}

// queueReload queues a file for reload with debouncing.
func (h *HotLoader) queueReload(filePath string) {
	h.debounceMu.Lock()
	h.pendingReloads[filePath] = time.Now()
	h.debounceMu.Unlock()
}

// debounceLoop processes pending reloads after the debounce delay.
func (h *HotLoader) debounceLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.processPendingReloads()
		}
	}
}

// processPendingReloads reloads files that have been pending for longer than debounceDelay.
func (h *HotLoader) processPendingReloads() {
	h.debounceMu.Lock()
	now := time.Now()
	var toReload []string
	for path, queuedAt := range h.pendingReloads {
		if now.Sub(queuedAt) >= h.debounceDelay {
			toReload = append(toReload, path)
			delete(h.pendingReloads, path)
		}
	}
	h.debounceMu.Unlock()

	for _, path := range toReload {
		h.reloadFile(path)
	}
}

// reloadFile reloads a viewdef file and pushes to sessions that have received it.
func (h *HotLoader) reloadFile(filePath string) {
	// Resolve the actual file to reload (handles symlink targets)
	reloadPath := h.resolveReloadPath(filePath)
	if reloadPath == "" {
		return
	}

	// Check if file exists (might have been deleted)
	info, err := os.Stat(reloadPath)
	if err != nil {
		h.config.Log(2, "ViewdefHotLoader: file not found %s", reloadPath)
		return
	}

	// Read the file content
	content, err := os.ReadFile(reloadPath)
	if err != nil {
		h.config.Log(1, "ViewdefHotLoader: error reading %s: %v", reloadPath, err)
		return
	}

	// Get the viewdef key from filename
	filename := filepath.Base(reloadPath)
	key := strings.TrimSuffix(filename, ".html")

	h.config.Log(1, "ViewdefHotLoader: reloading %s", key)

	// Update the viewdef in the manager
	h.manager.updateViewdef(key, string(content), reloadPath, info.ModTime())

	// Find sessions that have received this viewdef and push to them
	for _, sessionID := range h.sessions.GetSessionIDs() {
		if h.manager.hasSessionReceivedViewdef(sessionID, key) {
			// Push the updated viewdef to this session
			viewdefs := map[string]string{key: string(content)}
			h.sessions.PushViewdefs(sessionID, viewdefs)
			h.config.Log(2, "ViewdefHotLoader: pushed %s to session %s", key, sessionID)
		}
	}
}

// scanSymlinks scans the viewdef directory for symlinks and watches their target directories.
func (h *HotLoader) scanSymlinks() error {
	entries, err := os.ReadDir(h.viewdefDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		filePath := filepath.Join(h.viewdefDir, entry.Name())
		h.updateSymlinkWatch(filePath)
	}

	return nil
}

// updateSymlinkWatch checks if a file is a symlink and updates watches accordingly.
func (h *HotLoader) updateSymlinkWatch(filePath string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if file is a symlink
	info, err := os.Lstat(filePath)
	if err != nil {
		return
	}

	// Remove old watch if this file was previously a symlink
	if oldTarget, ok := h.symlinkTargets[filePath]; ok {
		h.removeWatchLocked(oldTarget)
		delete(h.symlinkTargets, filePath)
	}

	// If it's a symlink, resolve and watch the target directory
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(filePath)
		if err != nil {
			h.config.Log(2, "ViewdefHotLoader: cannot resolve symlink %s: %v", filePath, err)
			return
		}

		targetDir := filepath.Dir(target)
		h.symlinkTargets[filePath] = targetDir
		h.addWatchLocked(targetDir)
		h.config.Log(2, "ViewdefHotLoader: watching symlink target dir %s for %s", targetDir, filePath)
	}
}

// removeSymlinkWatch removes the watch for a symlink's target directory.
func (h *HotLoader) removeSymlinkWatch(filePath string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if targetDir, ok := h.symlinkTargets[filePath]; ok {
		h.removeWatchLocked(targetDir)
		delete(h.symlinkTargets, filePath)
	}
}

// addWatch adds a directory to the watch list.
func (h *HotLoader) addWatch(dir string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.addWatchLocked(dir)
}

func (h *HotLoader) addWatchLocked(dir string) error {
	h.watchedDirs[dir]++
	if h.watchedDirs[dir] == 1 {
		if err := h.watcher.Add(dir); err != nil {
			h.watchedDirs[dir]--
			return err
		}
		h.config.Log(2, "ViewdefHotLoader: added watch for %s", dir)
	}
	return nil
}

func (h *HotLoader) removeWatchLocked(dir string) {
	h.watchedDirs[dir]--
	if h.watchedDirs[dir] <= 0 {
		h.watcher.Remove(dir)
		delete(h.watchedDirs, dir)
		h.config.Log(2, "ViewdefHotLoader: removed watch for %s", dir)
	}
}

// resolveReloadPath determines which file to reload based on the changed path.
func (h *HotLoader) resolveReloadPath(changedPath string) string {
	// If the change is directly in the viewdef directory, use it
	if filepath.Dir(changedPath) == h.viewdefDir {
		// Check if file exists (might have been deleted)
		if _, err := os.Stat(changedPath); err != nil {
			return ""
		}
		return changedPath
	}

	// Otherwise, this is a change in a symlink target directory
	// Find which viewdef file symlinks to this location
	h.mu.Lock()
	defer h.mu.Unlock()

	changedDir := filepath.Dir(changedPath)
	changedBase := filepath.Base(changedPath)

	for viewdefPath, targetDir := range h.symlinkTargets {
		if targetDir == changedDir {
			// Check if the symlink points to this specific file
			target, err := filepath.EvalSymlinks(viewdefPath)
			if err == nil && filepath.Base(target) == changedBase {
				return viewdefPath
			}
		}
	}

	return ""
}
