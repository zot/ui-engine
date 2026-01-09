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
		pendingReloads: make(map[string]time.Time),
		debounceDelay:  100 * time.Millisecond,
		done:           make(chan struct{}),
	}

	return h, nil
}

// Start begins watching for file changes.
func (h *HotLoader) Start() error {
	// Watch the viewdef directory
	if err := h.watcher.Add(h.viewdefDir); err != nil {
		return err
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
	// Check if file exists (might have been deleted)
	info, err := os.Stat(filePath)
	if err != nil {
		h.config.Log(2, "ViewdefHotLoader: file not found %s", filePath)
		return
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		h.config.Log(1, "ViewdefHotLoader: error reading %s: %v", filePath, err)
		return
	}

	// Get the viewdef key from filename
	filename := filepath.Base(filePath)
	key := strings.TrimSuffix(filename, ".html")

	h.config.Log(1, "ViewdefHotLoader: reloading %s", key)

	// Update the viewdef in the manager
	h.manager.updateViewdef(key, string(content), filePath, info.ModTime())

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
