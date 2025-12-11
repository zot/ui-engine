// CRC: crc-MemoryStorage.md
// Spec: deployment.md
package storage

import (
	"fmt"
	"sync"
)

// MemoryStorage is an in-memory storage backend.
type MemoryStorage struct {
	variables  map[int64]*VariableData
	childIndex map[int64][]int64 // parentID -> childIDs
	mu         sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage backend.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		variables:  make(map[int64]*VariableData),
		childIndex: make(map[int64][]int64),
	}
}

// Store persists a variable to memory.
func (m *MemoryStorage) Store(v *VariableData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if this is an update (existing variable)
	existing, exists := m.variables[v.ID]

	// Update child index if parent changed
	if exists && existing.ParentID != v.ParentID {
		m.removeFromChildIndex(existing.ParentID, v.ID)
	}

	// Store the variable
	m.variables[v.ID] = v

	// Add to child index
	if v.ParentID != 0 && (!exists || existing.ParentID != v.ParentID) {
		m.childIndex[v.ParentID] = append(m.childIndex[v.ParentID], v.ID)
	}

	return nil
}

// Load retrieves a variable from memory.
func (m *MemoryStorage) Load(id int64) (*VariableData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.variables[id]
	if !ok {
		return nil, fmt.Errorf("variable %d not found", id)
	}

	// Return a copy
	return &VariableData{
		ID:         v.ID,
		ParentID:   v.ParentID,
		Value:      v.Value,
		Properties: copyProps(v.Properties),
		Unbound:    v.Unbound,
	}, nil
}

// Delete removes a variable from memory.
func (m *MemoryStorage) Delete(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok := m.variables[id]
	if !ok {
		return nil // Already deleted
	}

	// Remove from child index
	if v.ParentID != 0 {
		m.removeFromChildIndex(v.ParentID, id)
	}

	// Delete children recursively
	childIDs := m.childIndex[id]
	delete(m.childIndex, id)

	// Delete the variable
	delete(m.variables, id)

	// Recursively delete children (without lock - we're in a lock already)
	for _, childID := range childIDs {
		m.deleteRecursive(childID)
	}

	return nil
}

// deleteRecursive deletes a variable and its children (must be called with lock held).
func (m *MemoryStorage) deleteRecursive(id int64) {
	childIDs := m.childIndex[id]
	delete(m.childIndex, id)
	delete(m.variables, id)

	for _, childID := range childIDs {
		m.deleteRecursive(childID)
	}
}

// removeFromChildIndex removes a child from its parent's child index.
func (m *MemoryStorage) removeFromChildIndex(parentID, childID int64) {
	children := m.childIndex[parentID]
	for i, id := range children {
		if id == childID {
			m.childIndex[parentID] = append(children[:i], children[i+1:]...)
			break
		}
	}
	if len(m.childIndex[parentID]) == 0 {
		delete(m.childIndex, parentID)
	}
}

// LoadChildren gets all child variables of a parent.
func (m *MemoryStorage) LoadChildren(parentID int64) ([]*VariableData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	childIDs := m.childIndex[parentID]
	children := make([]*VariableData, 0, len(childIDs))

	for _, id := range childIDs {
		if v, ok := m.variables[id]; ok {
			children = append(children, &VariableData{
				ID:         v.ID,
				ParentID:   v.ParentID,
				Value:      v.Value,
				Properties: copyProps(v.Properties),
				Unbound:    v.Unbound,
			})
		}
	}

	return children, nil
}

// Exists checks if a variable exists.
func (m *MemoryStorage) Exists(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.variables[id]
	return ok
}

// Clear removes all data.
func (m *MemoryStorage) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.variables = make(map[int64]*VariableData)
	m.childIndex = make(map[int64][]int64)
	return nil
}

// BeginTransaction starts an atomic operation.
func (m *MemoryStorage) BeginTransaction() (Transaction, error) {
	return &memoryTransaction{
		storage:  m,
		stores:   make([]*VariableData, 0),
		deletes:  make([]int64, 0),
		committed: false,
	}, nil
}

// Close closes the storage backend.
func (m *MemoryStorage) Close() error {
	return nil
}

// Count returns the number of stored variables.
func (m *MemoryStorage) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.variables)
}

// copyProps creates a copy of a properties map.
func copyProps(props map[string]string) map[string]string {
	if props == nil {
		return nil
	}
	cp := make(map[string]string, len(props))
	for k, v := range props {
		cp[k] = v
	}
	return cp
}

// memoryTransaction implements Transaction for MemoryStorage.
type memoryTransaction struct {
	storage   *MemoryStorage
	stores    []*VariableData
	deletes   []int64
	committed bool
}

// Store queues a variable to be stored.
func (tx *memoryTransaction) Store(v *VariableData) error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	tx.stores = append(tx.stores, v)
	return nil
}

// Delete queues a variable to be deleted.
func (tx *memoryTransaction) Delete(id int64) error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	tx.deletes = append(tx.deletes, id)
	return nil
}

// Commit applies all queued operations.
func (tx *memoryTransaction) Commit() error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}
	tx.committed = true

	// Apply stores
	for _, v := range tx.stores {
		if err := tx.storage.Store(v); err != nil {
			return err
		}
	}

	// Apply deletes
	for _, id := range tx.deletes {
		if err := tx.storage.Delete(id); err != nil {
			return err
		}
	}

	return nil
}

// Rollback discards all queued operations.
func (tx *memoryTransaction) Rollback() error {
	tx.committed = true
	tx.stores = nil
	tx.deletes = nil
	return nil
}
