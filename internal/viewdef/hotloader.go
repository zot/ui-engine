// Package viewdef provides viewdef hot-loading support.
// CRC: crc-ViewdefStore.md | Seq: seq-viewdef-hotload.md, seq-app-discovery.md
// Spec: viewdefs.md, app-discovery.md
package viewdef

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

// SessionPusher provides session management for hot-loading viewdefs.
// CRC: crc-ViewdefStore.md
type SessionPusher interface {
	// GetSessionIDs returns a list of active vended session IDs.
	GetSessionIDs() []string
	// PushViewdefs pushes updated viewdefs to a session.
	PushViewdefs(sessionID string, viewdefs map[string]string)
}

// HotLoader watches lua/ for app activation and every viewdef source for
// file changes. Sources are watched through their lua/<app> entry, so events
// arrive under the path the file was loaded from (see cross-cutting concern
// "Hot-Loading Through Activation Entries").
// CRC: crc-ViewdefStore.md
type HotLoader struct {
	config   *config.Config
	luaDir   string
	watcher  *fsnotify.Watcher
	manager  *ViewdefManager
	sessions SessionPusher

	watchedSources map[string]bool // source dir -> watched
	mu             sync.Mutex

	// Debouncing
	pendingReloads map[string]time.Time
	debounceMu     sync.Mutex
	debounceDelay  time.Duration

	done chan struct{}
}

// NewHotLoader creates a hot loader for the site in cfg.Server.Dir.
func NewHotLoader(cfg *config.Config, manager *ViewdefManager, sessions SessionPusher) (*HotLoader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &HotLoader{
		config:         cfg,
		luaDir:         filepath.Join(cfg.Server.Dir, "lua"),
		watcher:        watcher,
		manager:        manager,
		sessions:       sessions,
		watchedSources: make(map[string]bool),
		pendingReloads: make(map[string]time.Time),
		debounceDelay:  100 * time.Millisecond,
		done:           make(chan struct{}),
	}, nil
}

// Start watches lua/ and every current viewdef source. A site viewdefs
// directory absent now is not watched for later (R131).
// CRC: crc-ViewdefStore.md | R129, R131, R132, R133
func (h *HotLoader) Start() error {
	if err := h.watcher.Add(h.luaDir); err != nil {
		h.config.Log(2, "ViewdefHotLoader: not watching %s: %v", h.luaDir, err)
	}
	h.mu.Lock()
	h.watchSourcesLocked(h.manager.Sources())
	h.mu.Unlock()

	go h.eventLoop()
	go h.debounceLoop()
	h.config.Log(1, "ViewdefHotLoader: watching %d viewdef sources", len(h.watchedSources))
	return nil
}

// Stop stops the hot loader.
func (h *HotLoader) Stop() error {
	close(h.done)
	return h.watcher.Close()
}

// watchSourcesLocked makes the watched set match sources.
// Seq: seq-app-discovery.md#2.1.2
func (h *HotLoader) watchSourcesLocked(sources []site.Source) {
	want := make(map[string]bool, len(sources))
	for _, src := range sources {
		want[src.Dir] = true
		if !h.watchedSources[src.Dir] {
			if err := h.watcher.Add(src.Dir); err != nil {
				h.config.Log(2, "ViewdefHotLoader: could not watch %s: %v", src.Dir, err)
				continue
			}
			h.watchedSources[src.Dir] = true
		}
	}
	for dir := range h.watchedSources {
		if !want[dir] {
			h.watcher.Remove(dir)
			delete(h.watchedSources, dir)
		}
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
			h.config.Log(1, "ViewdefHotLoader: watcher error: %v", err)
		}
	}
}

// handleEvent rediscovers on an app entry change in lua/, and queues a
// reload for a changed .html file.
// CRC: crc-ViewdefStore.md | Seq: seq-app-discovery.md#2.1 | R129
func (h *HotLoader) handleEvent(event fsnotify.Event) {
	h.config.Log(3, "ViewdefHotLoader: event %s on %s", event.Op, event.Name)
	if filepath.Dir(event.Name) == h.luaDir {
		if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
			h.rediscover()
		}
		return
	}
	if strings.HasSuffix(event.Name, ".html") && event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
		h.queueReload(event.Name)
	}
}

// rediscover re-runs app discovery, re-registers viewdefs, re-pushes the
// keys whose content changed, and updates the source watches.
// CRC: crc-ViewdefStore.md | Seq: seq-app-discovery.md#2.1 | R129
func (h *HotLoader) rediscover() {
	siteDir := h.config.Server.Dir
	sources := site.ViewdefSources(siteDir, site.Discover(siteDir, h.config.Server.BuiltinApps))
	changed := h.manager.SetSources(sources)
	h.mu.Lock()
	h.watchSourcesLocked(sources)
	h.mu.Unlock()
	for key, content := range changed {
		h.push(key, content)
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

// reloadFile reloads a viewdef file and pushes it to sessions that have
// received it. A file whose key a higher-ranked source owns is ignored.
// Seq: seq-viewdef-hotload.md
func (h *HotLoader) reloadFile(filePath string) {
	if _, err := os.Stat(filePath); err != nil {
		h.config.Log(2, "ViewdefHotLoader: file not found %s", filePath)
		return
	}
	key, content, ok := h.manager.ApplyFileChange(filePath)
	if !ok {
		return
	}
	h.config.Log(1, "ViewdefHotLoader: reloading %s", key)
	h.push(key, content)
}

// push sends one viewdef to every session that has received it.
func (h *HotLoader) push(key, content string) {
	for _, sessionID := range h.sessions.GetSessionIDs() {
		if h.manager.hasSessionReceivedViewdef(sessionID, key) {
			h.sessions.PushViewdefs(sessionID, map[string]string{key: content})
			h.config.Log(2, "ViewdefHotLoader: pushed %s to session %s", key, sessionID)
		}
	}
}
