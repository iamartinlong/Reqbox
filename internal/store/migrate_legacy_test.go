package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// The pre-rename (apimock) build shipped schema version 1, whose history table
// has no `timing` column. Upgrading has to add it without touching the rows,
// and this test builds that exact shape rather than trusting the current
// migrations slice to describe the past.
//
// Written after verifying by hand against a database produced by the real
// apimock binary: user_version was 1 and history lacked timing. A test that
// started from the current schema would prove nothing, because migration 1
// uses CREATE TABLE IF NOT EXISTS and would never recreate the old table.
func TestMigrateFromLegacyV1Database(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apimock.db")

	// Build a v1 database the way the old binary left it.
	legacy, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	const v1Schema = `
CREATE TABLE collections (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE requests (
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
CREATE TABLE history (
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
CREATE TABLE environments (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE env_vars (
	id TEXT PRIMARY KEY,
	environment_id TEXT NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
	key TEXT NOT NULL DEFAULT '',
	value TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	secret INTEGER NOT NULL DEFAULT 0,
	sort_index INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL DEFAULT ''
);
`
	if _, err := legacy.Exec(v1Schema); err != nil {
		t.Fatalf("create v1 schema: %v", err)
	}
	if _, err := legacy.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatalf("set version: %v", err)
	}
	if _, err := legacy.Exec(
		`INSERT INTO requests (id, name, method, url, expose_slug, exposed, created_at, updated_at)
		 VALUES ('r1', 'legacy request', 'GET', 'https://example.com/x', 'users', 1, '2026-10-20T00:00:00Z', '2026-10-20T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed request: %v", err)
	}
	if _, err := legacy.Exec(
		`INSERT INTO history (id, method, url, status, created_at)
		 VALUES ('h1', 'GET', 'https://example.com/x', 200, '2026-10-20T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed history: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	// Now open it with the current code, which should migrate in place.
	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open on a v1 database: %v", err)
	}
	defer st.Close()

	var version int
	if err := st.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != len(migrations) {
		t.Errorf("user_version = %d, want %d", version, len(migrations))
	}

	// The column the migration exists to add.
	var timing string
	if err := st.db.QueryRow("SELECT timing FROM history WHERE id = 'h1'").Scan(&timing); err != nil {
		t.Fatalf("history.timing is missing after migration: %v", err)
	}
	if timing != "" {
		t.Errorf("old rows should default timing to empty, got %q", timing)
	}

	// The old data has to survive untouched.
	reqs, err := st.ListRequests()
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].Name != "legacy request" {
		t.Fatalf("requests after migration: %+v", reqs)
	}
	if reqs[0].ExposeSlug != "users" || !reqs[0].Exposed {
		t.Errorf("expose fields were lost: slug=%q exposed=%v", reqs[0].ExposeSlug, reqs[0].Exposed)
	}

	// And the published mock must still be reachable by slug.
	bySlug, err := st.GetRequestBySlug("users")
	if err != nil {
		t.Fatalf("GetRequestBySlug after migration: %v", err)
	}
	if bySlug.ID != "r1" {
		t.Errorf("slug lookup returned %q", bySlug.ID)
	}

	hist, err := st.ListHistory("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || hist[0].Status != 200 {
		t.Fatalf("history after migration: %+v", hist)
	}
}

// Opening an already-migrated database a second time must be a no-op. The
// migration block is an ALTER TABLE, which fails outright if it runs twice;
// this is the guard against a user_version bookkeeping bug turning every
// restart into a startup failure.
func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reqbox.db")

	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		st, err := Open(path)
		if err != nil {
			t.Fatalf("reopen %d: %v", i+1, err)
		}
		if err := st.Close(); err != nil {
			t.Fatalf("close %d: %v", i+1, err)
		}
	}
}
