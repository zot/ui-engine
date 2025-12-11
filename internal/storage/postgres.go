// CRC: crc-PostgresStorage.md
// Spec: deployment.md
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresStorage is a PostgreSQL storage backend.
type PostgresStorage struct {
	db        *sql.DB
	tableName string
}

// NewPostgresStorage creates a new PostgreSQL storage backend.
// url should be a PostgreSQL connection string, e.g.:
// "postgres://user:password@localhost/dbname?sslmode=disable"
func NewPostgresStorage(url string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	s := &PostgresStorage{db: db, tableName: "variables"}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// migrate creates or updates the database schema.
func (s *PostgresStorage) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS variables (
			id BIGINT PRIMARY KEY,
			parent_id BIGINT DEFAULT 0,
			value JSONB,
			properties JSONB DEFAULT '{}',
			unbound BOOLEAN DEFAULT FALSE
		);
		CREATE INDEX IF NOT EXISTS idx_variables_parent_id ON variables(parent_id);
	`)
	return err
}

// Store persists a variable to PostgreSQL using INSERT ON CONFLICT UPDATE.
func (s *PostgresStorage) Store(v *VariableData) error {
	valueJSON, err := json.Marshal(v.Value)
	if err != nil {
		valueJSON = []byte("null")
	}

	propsJSON, err := json.Marshal(v.Properties)
	if err != nil {
		propsJSON = []byte("{}")
	}

	_, err = s.db.Exec(`
		INSERT INTO variables (id, parent_id, value, properties, unbound)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			parent_id = EXCLUDED.parent_id,
			value = EXCLUDED.value,
			properties = EXCLUDED.properties,
			unbound = EXCLUDED.unbound
	`, v.ID, v.ParentID, string(valueJSON), string(propsJSON), v.Unbound)

	return err
}

// Load retrieves a variable from PostgreSQL.
func (s *PostgresStorage) Load(id int64) (*VariableData, error) {
	var parentID int64
	var valueStr, propsStr sql.NullString
	var unbound bool

	err := s.db.QueryRow(`
		SELECT parent_id, value, properties, unbound
		FROM variables WHERE id = $1
	`, id).Scan(&parentID, &valueStr, &propsStr, &unbound)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("variable %d not found", id)
	}
	if err != nil {
		return nil, err
	}

	v := &VariableData{
		ID:       id,
		ParentID: parentID,
		Unbound:  unbound,
	}

	if valueStr.Valid && valueStr.String != "" && valueStr.String != "null" {
		v.Value = json.RawMessage(valueStr.String)
	}

	if propsStr.Valid && propsStr.String != "" && propsStr.String != "{}" {
		if err := json.Unmarshal([]byte(propsStr.String), &v.Properties); err != nil {
			v.Properties = make(map[string]string)
		}
	}

	return v, nil
}

// Delete removes a variable from PostgreSQL.
func (s *PostgresStorage) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM variables WHERE id = $1", id)
	return err
}

// LoadChildren gets all child variables of a parent.
func (s *PostgresStorage) LoadChildren(parentID int64) ([]*VariableData, error) {
	rows, err := s.db.Query(`
		SELECT id, parent_id, value, properties, unbound
		FROM variables WHERE parent_id = $1
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []*VariableData
	for rows.Next() {
		var id, pid int64
		var valueStr, propsStr sql.NullString
		var unbound bool

		if err := rows.Scan(&id, &pid, &valueStr, &propsStr, &unbound); err != nil {
			continue
		}

		v := &VariableData{
			ID:       id,
			ParentID: pid,
			Unbound:  unbound,
		}

		if valueStr.Valid && valueStr.String != "" && valueStr.String != "null" {
			v.Value = json.RawMessage(valueStr.String)
		}

		if propsStr.Valid && propsStr.String != "" && propsStr.String != "{}" {
			if err := json.Unmarshal([]byte(propsStr.String), &v.Properties); err != nil {
				v.Properties = make(map[string]string)
			}
		}

		children = append(children, v)
	}

	return children, rows.Err()
}

// Exists checks if a variable exists.
func (s *PostgresStorage) Exists(id int64) bool {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM variables WHERE id = $1)", id).Scan(&exists)
	return err == nil && exists
}

// Clear removes all data (uses TRUNCATE for efficiency).
func (s *PostgresStorage) Clear() error {
	_, err := s.db.Exec("TRUNCATE TABLE variables")
	return err
}

// BeginTransaction starts an atomic operation.
func (s *PostgresStorage) BeginTransaction() (Transaction, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return &postgresTransaction{tx: tx}, nil
}

// Close closes the storage backend.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// postgresTransaction implements Transaction for PostgreSQL.
type postgresTransaction struct {
	tx *sql.Tx
}

// Store persists a variable within the transaction.
func (t *postgresTransaction) Store(v *VariableData) error {
	valueJSON, _ := json.Marshal(v.Value)
	propsJSON, _ := json.Marshal(v.Properties)

	_, err := t.tx.Exec(`
		INSERT INTO variables (id, parent_id, value, properties, unbound)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			parent_id = EXCLUDED.parent_id,
			value = EXCLUDED.value,
			properties = EXCLUDED.properties,
			unbound = EXCLUDED.unbound
	`, v.ID, v.ParentID, string(valueJSON), string(propsJSON), v.Unbound)

	return err
}

// Delete removes a variable within the transaction.
func (t *postgresTransaction) Delete(id int64) error {
	_, err := t.tx.Exec("DELETE FROM variables WHERE id = $1", id)
	return err
}

// Commit completes the transaction.
func (t *postgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback cancels the transaction.
func (t *postgresTransaction) Rollback() error {
	return t.tx.Rollback()
}
