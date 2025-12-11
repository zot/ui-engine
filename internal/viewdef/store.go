// CRC: crc-ViewdefStore.md
// Spec: viewdefs.md
package viewdef

import (
	"encoding/json"
	"io/fs"
	"path"
	"strings"
	"sync"
)

// Store manages viewdef storage and delivery.
type Store struct {
	viewdefs       map[string]*Viewdef // TYPE.NAMESPACE -> Viewdef
	pendingUpdates map[string]string   // Batched updates: TYPE.NAMESPACE -> content
	sentTypes      map[string]bool     // Types whose viewdefs have been sent
	mu             sync.RWMutex
}

// NewStore creates a new viewdef store.
func NewStore() *Store {
	return &Store{
		viewdefs:       make(map[string]*Viewdef),
		pendingUpdates: make(map[string]string),
		sentTypes:      make(map[string]bool),
	}
}

// Store adds or replaces a viewdef.
func (s *Store) Store(viewdef *Viewdef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.viewdefs[viewdef.Key()] = viewdef
}

// Get retrieves a viewdef by TYPE.NAMESPACE.
// Falls back to TYPE.DEFAULT if TYPE.NAMESPACE not found.
func (s *Store) Get(typeName, namespace string) *Viewdef {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := typeName + "." + namespace
	if v, ok := s.viewdefs[key]; ok {
		return v
	}

	// Fallback to DEFAULT namespace
	if namespace != "DEFAULT" {
		defaultKey := typeName + ".DEFAULT"
		if v, ok := s.viewdefs[defaultKey]; ok {
			return v
		}
	}

	return nil
}

// GetByKey retrieves a viewdef by TYPE.NAMESPACE key.
func (s *Store) GetByKey(key string) *Viewdef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.viewdefs[key]
}

// Has checks if a viewdef exists.
func (s *Store) Has(typeName, namespace string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := typeName + "." + namespace
	_, ok := s.viewdefs[key]
	return ok
}

// GetForType returns all viewdefs for a type.
func (s *Store) GetForType(typeName string) []*Viewdef {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Viewdef
	prefix := typeName + "."
	for key, v := range s.viewdefs {
		if strings.HasPrefix(key, prefix) {
			result = append(result, v)
		}
	}
	return result
}

// Remove deletes a viewdef.
func (s *Store) Remove(typeName, namespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := typeName + "." + namespace
	delete(s.viewdefs, key)
}

// QueueForType queues all viewdefs for a type for delivery.
// Called when a variable of this type needs its viewdefs.
func (s *Store) QueueForType(typeName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Already sent this type's viewdefs
	if s.sentTypes[typeName] {
		return
	}

	prefix := typeName + "."
	for key, v := range s.viewdefs {
		if strings.HasPrefix(key, prefix) {
			s.pendingUpdates[key] = v.Content
		}
	}
	s.sentTypes[typeName] = true
}

// QueueViewdef queues a specific viewdef for delivery.
func (s *Store) QueueViewdef(viewdef *Viewdef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingUpdates[viewdef.Key()] = viewdef.Content
}

// FlushUpdates returns pending viewdef updates and clears the queue.
// Returns nil if no updates pending.
func (s *Store) FlushUpdates() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pendingUpdates) == 0 {
		return nil
	}

	updates := s.pendingUpdates
	s.pendingUpdates = make(map[string]string)
	return updates
}

// HasPendingUpdates checks if there are queued viewdef updates.
func (s *Store) HasPendingUpdates() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingUpdates) > 0
}

// GetPendingUpdatesJSON returns pending updates as JSON for variable 1's viewdefs property.
func (s *Store) GetPendingUpdatesJSON() ([]byte, error) {
	updates := s.FlushUpdates()
	if updates == nil {
		return nil, nil
	}
	return json.Marshal(updates)
}

// ResetSentTypes clears the record of sent types.
// Call when a new frontend connects.
func (s *Store) ResetSentTypes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sentTypes = make(map[string]bool)
}

// LoadFromFS loads viewdefs from a filesystem (e.g., embedded or os.DirFS).
// Expects files at html/viewdefs/TYPE.NAMESPACE.html
func (s *Store) LoadFromFS(fsys fs.FS) error {
	viewdefsDir := "html/viewdefs"

	entries, err := fs.ReadDir(fsys, viewdefsDir)
	if err != nil {
		// No viewdefs directory is OK
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".html") {
			continue
		}

		// Parse TYPE.NAMESPACE.html
		baseName := strings.TrimSuffix(name, ".html")
		typeName, namespace, err := ParseKey(baseName)
		if err != nil {
			continue // Skip invalid filenames
		}

		// Read content
		content, err := fs.ReadFile(fsys, path.Join(viewdefsDir, name))
		if err != nil {
			continue
		}

		// Store viewdef
		s.Store(NewViewdef(typeName, namespace, string(content)))
	}

	return nil
}

// Count returns the number of stored viewdefs.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.viewdefs)
}

// Keys returns all viewdef keys.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.viewdefs))
	for k := range s.viewdefs {
		keys = append(keys, k)
	}
	return keys
}
