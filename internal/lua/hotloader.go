// Package lua provides the Lua runtime and hot-loading support.
// CRC: crc-LuaHotLoader.md
// Spec: deployment.md
// Sequence: seq-lua-hotload.md
package lua

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zot/ui-engine/internal/config"
)

// HotLoader watches the lua directory for file changes and reloads modified files.
type HotLoader struct {
	config         *config.Config
	luaDir         string
	watcher        *fsnotify.Watcher
	getSessions    func() []*LuaSession       // Callback to get active sessions
	triggerRefresh func(sessionID string)     // Callback to trigger session refresh (runs AfterBatch)

	// Symlink tracking
	symlinkTargets map[string]string // lua file path -> resolved target dir
	watchedDirs    map[string]int    // dir path -> reference count
	mu             sync.Mutex

	// Debouncing
	pendingReloads map[string]time.Time
	debounceMu     sync.Mutex
	debounceDelay  time.Duration

	done chan struct{}
}

// NewHotLoader creates a new hot loader for the given lua directory.
// triggerRefresh is called after successful reload to run AfterBatch and push changes to browser.
func NewHotLoader(cfg *config.Config, luaDir string, getSessions func() []*LuaSession, triggerRefresh func(sessionID string)) (*HotLoader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	h := &HotLoader{
		config:         cfg,
		luaDir:         luaDir,
		watcher:        watcher,
		getSessions:    getSessions,
		triggerRefresh: triggerRefresh,
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
	// Watch the main lua directory
	if err := h.addWatch(h.luaDir); err != nil {
		return err
	}

	// Scan for existing symlinks and watch their target directories
	if err := h.scanSymlinks(); err != nil {
		h.config.Log(1, "HotLoader: error scanning symlinks: %v", err)
	}

	// Start the event loop
	go h.eventLoop()

	// Start the debounce processor
	go h.debounceLoop()

	h.config.Log(1, "HotLoader: watching %s for changes", h.luaDir)
	return nil
}

// Stop stops the hot loader.
func (h *HotLoader) Stop() error {
	close(h.done)
	return h.watcher.Close()
}

// scanSymlinks scans the lua directory for symlinks and watches their target directories.
func (h *HotLoader) scanSymlinks() error {
	entries, err := os.ReadDir(h.luaDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}
		filePath := filepath.Join(h.luaDir, entry.Name())
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
			h.config.Log(2, "HotLoader: cannot resolve symlink %s: %v", filePath, err)
			return
		}

		targetDir := filepath.Dir(target)
		h.symlinkTargets[filePath] = targetDir
		h.addWatchLocked(targetDir)
		h.config.Log(2, "HotLoader: watching symlink target dir %s for %s", targetDir, filePath)
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
		h.config.Log(2, "HotLoader: added watch for %s", dir)
	}
	return nil
}

func (h *HotLoader) removeWatchLocked(dir string) {
	h.watchedDirs[dir]--
	if h.watchedDirs[dir] <= 0 {
		h.watcher.Remove(dir)
		delete(h.watchedDirs, dir)
		h.config.Log(2, "HotLoader: removed watch for %s", dir)
	}
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
			h.config.Log(1, "HotLoader: watcher error: %v", err)
		}
	}
}

// handleEvent processes a single file system event.
func (h *HotLoader) handleEvent(event fsnotify.Event) {
	// Only care about .lua files
	if !strings.HasSuffix(event.Name, ".lua") {
		return
	}

	h.config.Log(3, "HotLoader: event %s on %s", event.Op, event.Name)

	// Handle symlink changes in the lua directory
	if filepath.Dir(event.Name) == h.luaDir {
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

// reloadFile reloads a Lua file in all active sessions.
// Wraps execution in panic recovery to prevent crashing the server.
// Triggers session refresh after reload to push viewdef/variable changes.
func (h *HotLoader) reloadFile(filePath string) {
	// Resolve the actual file to reload
	reloadPath := h.resolveReloadPath(filePath)
	if reloadPath == "" {
		return
	}

	h.config.Log(1, "HotLoader: reloading %s", reloadPath)

	// Read the file content
	content, err := os.ReadFile(reloadPath)
	if err != nil {
		h.config.Log(1, "HotLoader: error reading %s: %v", reloadPath, err)
		return
	}

	// Get the filename for the code name
	codeName := filepath.Base(reloadPath)

	// Reload in all active sessions with panic recovery
	sessions := h.getSessions()
	for _, sess := range sessions {
		h.reloadInSession(sess, codeName, string(content))
	}
}

// reloadInSession reloads code in a single session with panic recovery.
// Only reloads files that have already been loaded by the session.
// Sets session.reloading flag during reload.
func (h *HotLoader) reloadInSession(sess *LuaSession, codeName, content string) {
	// Check if file has been loaded by this session (skip if not)
	if !sess.IsFileLoaded(codeName) {
		h.config.Log(2, "HotLoader: skipping %s in session %s (not loaded)", codeName, sess.ID)
		return
	}

	// Panic recovery to prevent crashing the server
	defer func() {
		if r := recover(); r != nil {
			h.config.Log(0, "HotLoader: PANIC reloading %s in session %s: %v", codeName, sess.ID, r)
		}
	}()

	// Set reloading flag before reload
	sess.SetReloading(true)
	defer sess.SetReloading(false)

	_, err := sess.LoadCode(codeName, content)
	if err != nil {
		h.config.Log(1, "HotLoader: error reloading %s in session %s: %v", codeName, sess.ID, err)
		return
	}

	h.config.Log(2, "HotLoader: reloaded %s in session %s", codeName, sess.ID)

	// Trigger session refresh to run AfterBatch and push changes to browser
	if h.triggerRefresh != nil {
		h.config.Log(1, "HotLoader: triggering refresh for session %s", sess.ID)
		h.triggerRefresh(sess.ID)
	} else {
		h.config.Log(1, "HotLoader: triggerRefresh is nil, cannot refresh session %s", sess.ID)
	}
}

// resolveReloadPath determines which file to reload based on the changed path.
func (h *HotLoader) resolveReloadPath(changedPath string) string {
	// If the change is directly in the lua directory, use it
	if filepath.Dir(changedPath) == h.luaDir {
		// Check if file exists (might have been deleted)
		if _, err := os.Stat(changedPath); err != nil {
			return ""
		}
		return changedPath
	}

	// Otherwise, this is a change in a symlink target directory
	// Find which lua file symlinks to this location
	h.mu.Lock()
	defer h.mu.Unlock()

	changedDir := filepath.Dir(changedPath)
	changedBase := filepath.Base(changedPath)

	for luaPath, targetDir := range h.symlinkTargets {
		if targetDir == changedDir {
			// Check if the symlink points to this specific file
			target, err := filepath.EvalSymlinks(luaPath)
			if err == nil && filepath.Base(target) == changedBase {
				return luaPath
			}
		}
	}

	return ""
}
