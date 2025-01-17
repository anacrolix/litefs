package testingutil

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// OpenSQLDB opens a connection to a SQLite database.
func OpenSQLDB(tb testing.TB, dsn string) *sql.DB {
	tb.Helper()

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		tb.Fatal(err)
	} else if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		tb.Fatal(err)
	}

	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Fatal(err)
		}
	})

	return db
}

// ReopenSQLDB closes the existing database connection and reopens it with the DSN.
func ReopenSQLDB(tb testing.TB, db **sql.DB, dsn string) {
	tb.Helper()

	if err := (*db).Close(); err != nil {
		tb.Fatal(err)
	}
	*db = OpenSQLDB(tb, dsn)
}

// RetryUntil calls fn every interval until it returns nil or timeout elapses.
func RetryUntil(tb testing.TB, interval, timeout time.Duration, fn func() error) {
	tb.Helper()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var err error
	for {
		select {
		case <-ticker.C:
			if err = fn(); err == nil {
				return
			}
		case <-timer.C:
			tb.Fatalf("timeout: %s", err)
		}
	}
}

// MustCopyDir recursively copies files from src directory to dst directory.
func MustCopyDir(tb testing.TB, src, dst string) {
	if err := os.MkdirAll(dst, 0755); err != nil {
		tb.Fatal(err)
	}

	ents, err := os.ReadDir(src)
	if err != nil {
		tb.Fatal(err)
	}
	for _, ent := range ents {
		fi, err := os.Stat(filepath.Join(src, ent.Name()))
		if err != nil {
			tb.Fatal(err)
		}

		// If it's a directory, copy recursively.
		if fi.IsDir() {
			MustCopyDir(tb, filepath.Join(src, ent.Name()), filepath.Join(dst, ent.Name()))
			continue
		}

		// If it's a file, open the source file.
		r, err := os.Open(filepath.Join(src, ent.Name()))
		if err != nil {
			tb.Fatal(err)
		}
		defer func() { _ = r.Close() }()

		// Create destination file.
		w, err := os.Create(filepath.Join(dst, ent.Name()))
		if err != nil {
			tb.Fatal(err)
		}
		defer func() { _ = w.Close() }()

		// Copy contents of file to destination.
		if _, err := io.Copy(w, r); err != nil {
			tb.Fatal(err)
		}

		// Release file handles.
		if err := r.Close(); err != nil {
			tb.Fatal(err)
		} else if err := w.Close(); err != nil {
			tb.Fatal(err)
		}
	}
}
