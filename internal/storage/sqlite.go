// CRC: crc-SQLiteStorage.md
// Spec: deployment.md
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage is a SQLite storage backend.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage backend.
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStorage{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// init creates the necessary tables.
func (s *SQLiteStorage) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS variables (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER DEFAULT 0,
			value TEXT,
			properties TEXT,
			unbound INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_parent_id ON variables(parent_id);
	`)
	return err
}

// Store persists a variable to SQLite.
func (s *SQLiteStorage) Store(v *VariableData) error {
	valueJSON, err := json.Marshal(v.Value)
	if err != nil {
		valueJSON = []byte("null")
	}

	propsJSON, err := json.Marshal(v.Properties)
	if err != nil {
		propsJSON = []byte("{}")
	}

	unboundInt := 0
	if v.Unbound {
		unboundInt = 1
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO variables (id, parent_id, value, properties, unbound)
		VALUES (?, ?, ?, ?, ?)
	`, v.ID, v.ParentID, string(valueJSON), string(propsJSON), unboundInt)

	return err
}

// Load retrieves a variable from SQLite.
func (s *SQLiteStorage) Load(id int64) (*VariableData, error) {
	var parentID int64
	var valueStr, propsStr string
	var unboundInt int

	err := s.db.QueryRow(`
		SELECT parent_id, value, properties, unbound
		FROM variables WHERE id = ?
	`, id).Scan(&parentID, &valueStr, &propsStr, &unboundInt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("variable %d not found", id)
	}
	if err != nil {
		return nil, err
	}

	v := &VariableData{
		ID:       id,
		ParentID: parentID,
		Unbound:  unboundInt != 0,
	}

	if valueStr != "" && valueStr != "null" {
		v.Value = json.RawMessage(valueStr)
	}

	if propsStr != "" && propsStr != "{}" {
		if err := json.Unmarshal([]byte(propsStr), &v.Properties); err != nil {
			v.Properties = make(map[string]string)
		}
	}

	return v, nil
}

// Delete removes a variable from SQLite.
func (s *SQLiteStorage) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM variables WHERE id = ?", id)
	return err
}

// LoadChildren gets all child variables of a parent.
func (s *SQLiteStorage) LoadChildren(parentID int64) ([]*VariableData, error) {
	rows, err := s.db.Query(`
		SELECT id, parent_id, value, properties, unbound
		FROM variables WHERE parent_id = ?
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []*VariableData
	for rows.Next() {
		var id, pid int64
		var valueStr, propsStr string
		var unboundInt int

		if err := rows.Scan(&id, &pid, &valueStr, &propsStr, &unboundInt); err != nil {
			continue
		}

		v := &VariableData{
			ID:       id,
			ParentID: pid,
			Unbound:  unboundInt != 0,
		}

		if valueStr != "" && valueStr != "null" {
			v.Value = json.RawMessage(valueStr)
		}

		if propsStr != "" && propsStr != "{}" {
			if err := json.Unmarshal([]byte(propsStr), &v.Properties); err != nil {
				v.Properties = make(map[string]string)
			}
		}

		children = append(children, v)
	}

	return children, rows.Err()
}

// Exists checks if a variable exists.
func (s *SQLiteStorage) Exists(id int64) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM variables WHERE id = ?", id).Scan(&count)
	return err == nil && count > 0
}

// Clear removes all data.
func (s *SQLiteStorage) Clear() error {
	_, err := s.db.Exec("DELETE FROM variables")
	return err
}

// BeginTransaction starts an atomic operation.
func (s *SQLiteStorage) BeginTransaction() (Transaction, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return &sqliteTransaction{tx: tx}, nil
}

// Close closes the storage backend.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// sqliteTransaction implements Transaction for SQLite.
type sqliteTransaction struct {
	tx *sql.Tx
}

// Store persists a variable within the transaction.
func (t *sqliteTransaction) Store(v *VariableData) error {
	valueJSON, _ := json.Marshal(v.Value)
	propsJSON, _ := json.Marshal(v.Properties)

	unboundInt := 0
	if v.Unbound {
		unboundInt = 1
	}

	_, err := t.tx.Exec(`
		INSERT OR REPLACE INTO variables (id, parent_id, value, properties, unbound)
		VALUES (?, ?, ?, ?, ?)
	`, v.ID, v.ParentID, string(valueJSON), string(propsJSON), unboundInt)

	return err
}

// Delete removes a variable within the transaction.
func (t *sqliteTransaction) Delete(id int64) error {
	_, err := t.tx.Exec("DELETE FROM variables WHERE id = ?", id)
	return err
}

// Commit completes the transaction.
func (t *sqliteTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback cancels the transaction.
func (t *sqliteTransaction) Rollback() error {
	return t.tx.Rollback()
}
