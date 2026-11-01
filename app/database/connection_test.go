package database

import (
	"context"
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"
)

func TestNewConnectionKeepsExistingDSNQuery(t *testing.T) {
	db, err := NewConnection("file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestPragmasSurviveReconnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := NewConnection(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	check := func(stage string) {
		t.Helper()
		var foreignKeys, busyTimeout int
		var journalMode string
		if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatal(err)
		}
		if foreignKeys != 1 || busyTimeout != 5000 || journalMode != "wal" {
			t.Errorf("%s: foreign_keys=%d busy_timeout=%d journal_mode=%s", stage, foreignKeys, busyTimeout, journalMode)
		}
	}
	check("first connection")

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Returning ErrBadConn makes database/sql discard this connection.
	conn.Raw(func(any) error { return driver.ErrBadConn })
	conn.Close()

	check("after reconnect")

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file is not at the plain path: %v", err)
	}
}
