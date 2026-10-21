package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTempDir(t *testing.T) {
	if !isTempDir(os.TempDir()) {
		t.Errorf("the temp root itself should count as temporary: %s", os.TempDir())
	}
	if !isTempDir(filepath.Join(os.TempDir(), "reqbox-ship")) {
		t.Error("a folder inside temp should count as temporary")
	}

	// The filesystem root is never inside temp, and unlike the working directory
	// it stays outside temp even when the repo itself was cloned into temp.
	root := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	if isTempDir(root) {
		t.Errorf("filesystem root %q should not count as temporary", root)
	}
}

func TestDefaultDBPathIsAbsolute(t *testing.T) {
	path := defaultDBPath()
	if path == "" {
		t.Fatal("defaultDBPath returned an empty path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected an absolute path, got %q", path)
	}
	if !strings.HasSuffix(strings.ToLower(path), ".db") {
		t.Errorf("expected a .db file, got %q", path)
	}
}

func TestDisplayURL(t *testing.T) {
	cases := map[string]string{
		":5178":          "http://localhost:5178",
		"0.0.0.0:8080":   "http://localhost:8080",
		"127.0.0.1:9000": "http://127.0.0.1:9000",
		"[::]:5178":      "http://localhost:5178",
		"[::1]:5178":     "http://[::1]:5178",
	}
	for in, want := range cases {
		if got := displayURL(in); got != want {
			t.Errorf("displayURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// The default is loopback, and anything that accepts remote connections should
// be recognised so the startup warning fires. Treating a wildcard as "safe"
// would keep the warning silent in exactly the case it exists for.
func TestListensOnLoopbackOnly(t *testing.T) {
	loopback := []string{
		"127.0.0.1:5178",
		"127.0.0.5:5178",
		"[::1]:5178",
		"localhost:5178",
	}
	for _, addr := range loopback {
		if !listensOnLoopbackOnly(addr) {
			t.Errorf("%q should be treated as loopback-only", addr)
		}
	}

	exposed := []string{
		":5178",
		"0.0.0.0:5178",
		"[::]:5178",
		"192.168.1.10:5178",
		"example.com:5178",
		"5178",
	}
	for _, addr := range exposed {
		if listensOnLoopbackOnly(addr) {
			t.Errorf("%q should NOT be treated as loopback-only", addr)
		}
	}
}

func TestEnvOr(t *testing.T) {
	const key = "REQBOX_TEST_VALUE"
	t.Setenv(key, "")
	if got := envOr(key, "fallback"); got != "fallback" {
		t.Errorf("empty env should fall back, got %q", got)
	}
	t.Setenv(key, "explicit")
	if got := envOr(key, "fallback"); got != "explicit" {
		t.Errorf("set env should win, got %q", got)
	}
}

// A deployment whose start script still sets APIMOCK_* must keep working after
// the rename, otherwise upgrading silently changes where data lives.
func TestEnvOrReadsLegacyName(t *testing.T) {
	t.Setenv("REQBOX_ADDR", "")
	t.Setenv("APIMOCK_ADDR", "127.0.0.1:9999")
	if got := envOr("REQBOX_ADDR", "fallback"); got != "127.0.0.1:9999" {
		t.Errorf("legacy variable should be honoured, got %q", got)
	}

	// Both set: the current name is the one that counts.
	t.Setenv("REQBOX_ADDR", "127.0.0.1:1111")
	if got := envOr("REQBOX_ADDR", "fallback"); got != "127.0.0.1:1111" {
		t.Errorf("current variable should win over legacy, got %q", got)
	}
}

// The rename changed the default database file name. An upgrade must find the
// existing data instead of starting empty.
func TestLegacyDBPathPrefersExistingOldFile(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "reqbox.db")
	oldPath := filepath.Join(dir, "apimock.db")

	// Only the old file exists: it should be reused.
	if err := os.WriteFile(oldPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := legacyDBPath(newPath); got != oldPath {
		t.Errorf("expected the old database, got %q", got)
	}

	// Once a new file exists it wins, so a user who deliberately starts fresh
	// does not get pulled back to the old data.
	if err := os.WriteFile(newPath, []byte("y"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := legacyDBPath(newPath); got != "" {
		t.Errorf("new file should win, got %q", got)
	}
}

func TestLegacyDBPathReturnsEmptyWhenNothingExists(t *testing.T) {
	dir := t.TempDir()
	if got := legacyDBPath(filepath.Join(dir, "reqbox.db")); got != "" {
		t.Errorf("nothing should be found, got %q", got)
	}
}
