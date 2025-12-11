// Package storage implements storage backends for the UI server.
// CRC: crc-StorageBackend.md
// Spec: deployment.md, data-models.md
package storage

import (
	"encoding/json"
)

// VariableData represents stored variable data.
type VariableData struct {
	ID         int64             `json:"id"`
	ParentID   int64             `json:"parentId,omitempty"`
	Value      json.RawMessage   `json:"value,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Unbound    bool              `json:"unbound,omitempty"`
}

// Backend defines the interface for storage backends.
type Backend interface {
	// Store persists a variable to storage.
	Store(v *VariableData) error

	// Load retrieves a variable from storage.
	Load(id int64) (*VariableData, error)

	// Delete removes a variable from storage.
	Delete(id int64) error

	// LoadChildren gets all child variables of a parent.
	LoadChildren(parentID int64) ([]*VariableData, error)

	// Exists checks if a variable exists.
	Exists(id int64) bool

	// Clear removes all data.
	Clear() error

	// BeginTransaction starts an atomic operation.
	BeginTransaction() (Transaction, error)

	// Close closes the storage backend.
	Close() error
}

// Transaction represents an atomic storage operation.
type Transaction interface {
	// Store persists a variable within the transaction.
	Store(v *VariableData) error

	// Delete removes a variable within the transaction.
	Delete(id int64) error

	// Commit completes the transaction.
	Commit() error

	// Rollback cancels the transaction.
	Rollback() error
}
