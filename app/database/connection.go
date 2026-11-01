package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

// querier is satisfied by *DB and *sql.Tx.
type querier interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

type rowsQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// connectionPragmas are in the DSN, not run once after opening: most are per
// connection, and database/sql can replace a connection it finds bad.
const connectionPragmas = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

func NewConnection(dbPath string) (*DB, error) {
	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}

	db, err := sql.Open("sqlite", dbPath+separator+connectionPragmas)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}
