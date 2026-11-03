// Package storage provides SQLite-backed persistence for paired workspaces.
package storage

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DB wraps a SQLite connection with Connect's domain queries.
type DB struct {
	sql *sql.DB
}

var ErrNotFound = errors.New("not found")

// Open opens (creating if needed) the SQLite database at path and applies
// the embedded schema.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}
	// SQLite only supports one writer at a time; a single connection avoids
	// SQLITE_BUSY errors under concurrent goroutines in this process.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("storage: apply schema: %w", err)
	}

	return &DB{sql: sqlDB}, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.sql.Close()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
