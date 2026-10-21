package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
	// PermissionWarning is non-nil when the database file could not be
	// restricted to its owner. It is reported at startup rather than treated as
	// a fatal error, because failing to open the database is worse than opening
	// it with permissions the user has been warned about.
	PermissionWarning error
}

var migrations = []string{
	// 0 -> 1
	`
CREATE TABLE IF NOT EXISTS collections (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS requests (
	id TEXT PRIMARY KEY,
	collection_id TEXT REFERENCES collections(id) ON DELETE SET NULL,
	name TEXT NOT NULL,
	method TEXT NOT NULL DEFAULT 'GET',
	url TEXT NOT NULL DEFAULT '',
	params TEXT NOT NULL DEFAULT '[]',
	headers TEXT NOT NULL DEFAULT '[]',
	body_type TEXT NOT NULL DEFAULT 'none',
	body TEXT NOT NULL DEFAULT '',
	form TEXT NOT NULL DEFAULT '[]',
	auth_type TEXT NOT NULL DEFAULT 'none',
	auth TEXT NOT NULL DEFAULT '{}',
	sort_index INTEGER NOT NULL DEFAULT 0,
	expose_slug TEXT NOT NULL DEFAULT '',
	exposed INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_requests_collection ON requests(collection_id, sort_index);
CREATE UNIQUE INDEX IF NOT EXISTS idx_requests_slug ON requests(expose_slug) WHERE expose_slug <> '';

CREATE TABLE IF NOT EXISTS history (
	id TEXT PRIMARY KEY,
	request_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL DEFAULT '',
	method TEXT NOT NULL DEFAULT '',
	url TEXT NOT NULL DEFAULT '',
	status INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	ok INTEGER NOT NULL DEFAULT 0,
	error TEXT NOT NULL DEFAULT '',
	request_snapshot TEXT NOT NULL DEFAULT '',
	response_headers TEXT NOT NULL DEFAULT '',
	response_body TEXT NOT NULL DEFAULT '',
	body_encoding TEXT NOT NULL DEFAULT 'text',
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_history_created ON history(created_at DESC);

CREATE TABLE IF NOT EXISTS environments (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS env_vars (
	id TEXT PRIMARY KEY,
	environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
	key TEXT NOT NULL DEFAULT '',
	value TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	secret INTEGER NOT NULL DEFAULT 0,
	sort_index INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_env_vars_env ON env_vars(environment_id, sort_index);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL DEFAULT ''
);
`,
	// 2: per-phase timing. Added as a new block rather than editing block 1,
	// because SQLite never re-runs a migration that has already been applied.
	// ALTER TABLE ... ADD COLUMN is safe here: existing rows get the default.
	`
ALTER TABLE history ADD COLUMN timing TEXT NOT NULL DEFAULT '';
`,
}

// Open opens (and creates) the database at path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path is empty")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		// 0o700 and a restrictive ACL: the database holds plaintext credentials
		// for every saved request, so another account on the same machine must
		// not be able to read it.
		if err := restrictDir(dir); err != nil {
			return nil, fmt.Errorf("secure data directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite allows one writer at a time; a small pool keeps concurrent readers happy.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	// SQLite creates the file itself, so the restriction has to be applied after
	// it exists. A failure here is reported but not fatal: an unreadable database
	// is worse than a permissive one, and the caller surfaces the warning.
	s.PermissionWarning = restrictToCurrentUser(path)
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		if _, err := s.db.Exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := s.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			return err
		}
	}
	return nil
}

func newID() string { return uuid.NewString() }

func now() Timestamp { return Timestamp{Time: time.Now().UTC()} }

func parseTime(s string) Timestamp {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Timestamp{}
	}
	return Timestamp{Time: t.UTC()}
}

func formatTime(t Timestamp) string { return t.UTC().Format(time.RFC3339Nano) }

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
