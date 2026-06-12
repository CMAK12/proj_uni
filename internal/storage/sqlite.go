// Package storage provides SQLite-backed implementations of the repository
// interfaces declared in package service. It is the concrete side of the
// Repository pattern: business logic depends on the interfaces, these types
// satisfy them, and main injects them — so the backend can be swapped without
// touching the service layer.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sqlite "modernc.org/sqlite" // also registers the pure-Go "sqlite" driver
	sqlitelib "modernc.org/sqlite/lib"
)

// schema is the full database schema. It is idempotent (IF NOT EXISTS) so Open
// can run it on every start.
const schema = `
CREATE TABLE IF NOT EXISTS students (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS courses (
    id         TEXT PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    credits    INTEGER NOT NULL,
    type       TEXT NOT NULL,
    grading    TEXT NOT NULL,
    features   TEXT NOT NULL,
    platform   TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS enrollments (
    id          TEXT PRIMARY KEY,
    student_id  TEXT NOT NULL REFERENCES students(id),
    course_id   TEXT NOT NULL REFERENCES courses(id),
    status      TEXT NOT NULL,
    final_grade REAL NOT NULL DEFAULT 0,
    letter      TEXT NOT NULL DEFAULT '',
    passed      INTEGER NOT NULL DEFAULT 0,
    progress    REAL NOT NULL DEFAULT 0,
    enrolled_at TEXT NOT NULL,
    UNIQUE(student_id, course_id)
);
CREATE TABLE IF NOT EXISTS assessments (
    id            TEXT PRIMARY KEY,
    enrollment_id TEXT NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    score         REAL NOT NULL,
    max_score     REAL NOT NULL,
    weight        REAL NOT NULL
);`

// Open opens (or creates) a SQLite database at dsn, enables foreign keys and
// applies the schema. Pass ":memory:" for an ephemeral database. The pool is
// capped to one connection: SQLite is a single-writer engine and this also keeps
// an in-memory database alive across queries.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// migrate executes each statement in the schema.
func migrate(db *sql.DB) error {
	for _, stmt := range strings.Split(schema, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE/PRIMARY KEY constraint
// failure, so repositories can translate it to domain.ErrDuplicate.
func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	code := se.Code()
	return code == sqlitelib.SQLITE_CONSTRAINT_UNIQUE ||
		code == sqlitelib.SQLITE_CONSTRAINT_PRIMARYKEY
}

// timeLayout is the textual representation used for all timestamps.
const timeLayout = "2006-01-02T15:04:05Z07:00"
