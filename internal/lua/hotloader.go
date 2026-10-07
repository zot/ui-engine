// Package lua provides the Lua runtime and hot-loading support.
// CRC: crc-LuaHotLoader.md | Seq: seq-lua-hotload.md, seq-unload-module.md, seq-app-discovery.md
// Spec: deployment.md, module-tracking.md, app-discovery.md
package lua

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/site"
)

// HotLoader watches the lua directory for file changes and reloads modified files.
// CRC: crc-LuaHotLoader.md
type HotLoader struct {
	config         *config.Config
	luaDir         string
	watcher        *fsnotify.Watcher
	getSessions    func() []*LuaSession   // Callback to get active sessions
	triggerRefresh func(sessionID string) // Callback to trigger session refresh (runs AfterBatch)

	// Watches. App directories are watched through their lua/<app> entry
	// (cross-cutting concern "Hot-Loading Through Activation Entries").
	appWatches  map[string][]string // app name -> directories watched under its entry
	watchedDirs map[string]int      // dir path -> reference count
	mu          sync.Mutex

	// Debouncing, keyed by tracking key so one file reached through two
	// watched paths (lua/<app>/ and apps/<app>/) reloads once (R134).
	pendingReloads map[string]pendingReload
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
		appWatches:     make(map[string][]string),
		watchedDirs:    make(map[string]int),
		pendingReloads: make(map[string]pendingReload),
		debounceDelay:  100 * time.Millisecond,
		done:           make(chan struct{}),
	}

	return h, nil
}

// pendingReload is a debounced reload: the path last seen and when.
type pendingReload struct {
	path     string
	queuedAt time.Time
}

// Start begins watching for file changes: lua/ itself, apps/, and the tree
// under each active app's entry.
// CRC: crc-LuaHotLoader.md | R132, R134
func (h *HotLoader) Start() error {
	// Watch the main lua directory
	if err := h.addWatch(h.luaDir); err != nil {
		return err
	}

	// Watch the apps directory if it exists
	appsDir := filepath.Join(h.config.Server.Dir, "apps")
	if info, err := os.Stat(appsDir); err == nil && info.IsDir() {
		if err := h.addWatchRecursive(appsDir); err != nil {
			h.config.Log(1, "HotLoader: error watching apps directory: %v", err)
		}
	}

	h.rediscover()

	// Start the event loop
	go h.eventLoop()

	// Start the debounce processor
	go h.debounceLoop()

	h.config.Log(1, "HotLoader: watching %s for changes", h.luaDir)
	return nil
}

// addWatchRecursive adds a watch to a directory and all its subdirectories.
func (h *HotLoader) addWatchRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}
		if info.IsDir() {
			if err := h.addWatch(path); err != nil {
				h.config.Log(2, "HotLoader: could not watch %s: %v", path, err)
			}
		}
		return nil
	})
}

// Stop stops the hot loader.
func (h *HotLoader) Stop() error {
	close(h.done)
	return h.watcher.Close()
}

// CleanupModule removes tracking state for a module file.
// CRC: crc-LuaHotLoader.md | R136
// Called when a module is unloaded via session:unloadModule().
// Seq: seq-unload-module.md
func (h *HotLoader) CleanupModule(trackingKey string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove pending reload if queued
	h.debounceMu.Lock()
	delete(h.pendingReloads, trackingKey)
	h.debounceMu.Unlock()
}

// CleanupDirectory removes tracking state for all files in a directory.
// CRC: crc-LuaHotLoader.md | R135
// Called when a directory is unloaded via session:unloadDirectory().
// Seq: seq-unload-module.md
func (h *HotLoader) CleanupDirectory(dirPath string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove pending reloads for files in this directory
	h.debounceMu.Lock()
	for key, pending := range h.pendingReloads {
		if inDir(key, dirPath) || inDir(pending.path, dirPath) {
			delete(h.pendingReloads, key)
		}
	}
	h.debounceMu.Unlock()

	// Remove watch if we're watching this specific directory
	if h.watchedDirs[dirPath] > 0 {
		h.removeWatchLocked(dirPath)
	}
}

// inDir reports whether path is dir or lies under it.
func inDir(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

// rediscover re-runs app discovery and makes the per-app watches match:
// new apps get their tree watched through the entry, removed apps lose
// theirs. Lua already loaded from a removed app stays loaded (R130).
// CRC: crc-LuaHotLoader.md | Seq: seq-app-discovery.md#2.2 | R129, R130, R132
func (h *HotLoader) rediscover() {
	apps := site.Discover(h.config.Server.Dir, h.config.Server.BuiltinApps)
	h.mu.Lock()
	defer h.mu.Unlock()
	active := make(map[string]bool, len(apps))
	for _, app := range apps {
		active[app.Name] = true
		if _, watched := h.appWatches[app.Name]; !watched {
			h.appWatches[app.Name] = h.watchTreeLocked(app.Dir)
		}
	}
	for name, dirs := range h.appWatches {
		if !active[name] {
			for _, dir := range dirs {
				h.removeWatchLocked(dir)
			}
			delete(h.appWatches, name)
		}
	}
}

// watchTreeLocked watches dir and its subdirectories, reading through dir
// itself (an entry symlink or junction) and skipping dot-directories.
// Returns the directories watched.
// Seq: seq-app-discovery.md#2.2.1
func (h *HotLoader) watchTreeLocked(dir string) []string {
	if err := h.addWatchLocked(dir); err != nil {
		h.config.Log(2, "HotLoader: could not watch %s: %v", dir, err)
		return nil
	}
	watched := []string{dir}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return watched
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			watched = append(watched, h.watchTreeLocked(filepath.Join(dir, entry.Name()))...)
		}
	}
	return watched
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

// handleEvent processes a single file system event: an app entry created or
// removed in lua/ re-runs discovery; a changed .lua file queues a reload.
// CRC: crc-LuaHotLoader.md | Seq: seq-app-discovery.md#2.2 | R129
func (h *HotLoader) handleEvent(event fsnotify.Event) {
	h.config.Log(3, "HotLoader: event %s on %s", event.Op, event.Name)

	isLua := strings.HasSuffix(event.Name, ".lua")
	if !isLua && filepath.Dir(event.Name) == h.luaDir && event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
		h.rediscover()
		return
	}
	if !isLua {
		return
	}

	// Queue reload for write events
	if event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0 {
		h.queueReload(event.Name)
	}
}

// queueReload queues a file for reload, debounced so the same file seen
// through two watched paths reloads once. Entries are merged by file
// identity, not only by tracking key: on Windows EvalSymlinks does not
// resolve junctions, so one file can carry two tracking keys there.
// CRC: crc-LuaHotLoader.md | R134
func (h *HotLoader) queueReload(filePath string) {
	key, err := ComputeTrackingKey(h.config.Server.Dir, filePath)
	if err != nil {
		key = filePath
	}
	h.debounceMu.Lock()
	defer h.debounceMu.Unlock()
	if info, err := os.Stat(filePath); err == nil {
		for pendingKey, pending := range h.pendingReloads {
			if pendingKey == key {
				break
			}
			if other, err := os.Stat(pending.path); err == nil && os.SameFile(info, other) {
				key = pendingKey
				break
			}
		}
	}
	h.pendingReloads[key] = pendingReload{path: filePath, queuedAt: time.Now()}
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
	for key, pending := range h.pendingReloads {
		if now.Sub(pending.queuedAt) >= h.debounceDelay {
			toReload = append(toReload, pending.path)
			delete(h.pendingReloads, key)
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
// Seq: seq-lua-hotload.md
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

	trackingKey, err := ComputeTrackingKey(h.config.Server.Dir, reloadPath)
	if err != nil {
		h.config.Log(1, "HotLoader: error computing tracking key for %s: %v", reloadPath, err)
		return
	}

	h.config.Log(2, "HotLoader: tracking key for %s is %s", reloadPath, trackingKey)

	// Reload in all active sessions with panic recovery
	sessions := h.getSessions()
	for _, sess := range sessions {
		h.reloadInSession(sess, trackingKey, string(content))
	}
}

// reloadInSession reloads code in a single session with panic recovery.
// Only reloads files that have already been loaded by the session.
// Sets session.reloading flag during reload.
// Seq: seq-lua-hotload.md
func (h *HotLoader) reloadInSession(sess *LuaSession, trackingKey, content string) {
	// Check if file has been loaded by this session (skip if not)
	if !sess.IsFileLoaded(trackingKey) {
		h.config.Log(2, "HotLoader: skipping %s in session %s (not loaded)", trackingKey, sess.ID)
		return
	}

	// Panic recovery to prevent crashing the server
	defer func() {
		if r := recover(); r != nil {
			h.config.Log(0, "HotLoader: PANIC reloading %s in session %s: %v", trackingKey, sess.ID, r)
		}
	}()

	// Set reloading flag before reload
	sess.SetReloading(true)
	defer sess.SetReloading(false)

	_, err := sess.LoadCode(trackingKey, content)
	if err != nil {
		h.config.Log(1, "HotLoader: error reloading %s in session %s: %v", trackingKey, sess.ID, err)
		return
	}

	h.config.Log(2, "HotLoader: reloaded %s in session %s", trackingKey, sess.ID)

	// Trigger session refresh to run AfterBatch and push changes to browser
	if h.triggerRefresh != nil {
		h.config.Log(1, "HotLoader: triggering refresh for session %s", sess.ID)
		h.triggerRefresh(sess.ID)
	} else {
		h.config.Log(1, "HotLoader: triggerRefresh is nil, cannot refresh session %s", sess.ID)
	}
}

// resolveReloadPath returns changedPath if it is an existing .lua file, else
// "". Watches run through the lua/<app> entry, so the path is already the one
// the file was loaded from; no reverse mapping is needed.
func (h *HotLoader) resolveReloadPath(changedPath string) string {
	if !strings.HasSuffix(changedPath, ".lua") {
		return ""
	}
	if _, err := os.Stat(changedPath); err != nil {
		return ""
	}
	return changedPath
}
