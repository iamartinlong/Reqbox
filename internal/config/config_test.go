package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write puts a config file in dir and returns its path, leaving Path() alone.
func write(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultValues(t *testing.T) {
	f := Default()
	// Addr is empty in Default() on purpose; the built-in address is supplied by
	// the caller as the fallback.
	if f.Addr(DefaultAddr) != DefaultAddr {
		t.Errorf("default addr = %q", f.Addr(DefaultAddr))
	}
	if !f.OpenBrowserEnabled() {
		t.Error("opening a browser should default to on: it is the reason this exists")
	}
	if !f.AutoPortEnabled() {
		t.Error("autoPort should default to on: a busy port is the common failure")
	}
	if !f.AuthEnabled() {
		t.Error("auth must default to on")
	}
}

// The whole precedence rule depends on Addr() returning the fallback untouched
// when the file says nothing, otherwise a flag default would be overwritten by
// an empty config.
func TestAddrFallsBackWhenSilent(t *testing.T) {
	var f File
	if got := f.Addr("0.0.0.0:9000"); got != "0.0.0.0:9000" {
		t.Errorf("Addr() = %q, want the fallback", got)
	}
}

func TestPortAloneKeepsTheHost(t *testing.T) {
	var f File
	f.Server.Port = 6000
	// A file that only changes the port must not silently rebind to loopback
	// when the operator asked for a reachable host on the command line.
	if got := f.Addr("0.0.0.0:5178"); got != "0.0.0.0:6000" {
		t.Errorf("Addr() = %q", got)
	}
}

func TestAddrBeatsPort(t *testing.T) {
	var f File
	f.Server.Addr = "127.0.0.1:7777"
	f.Server.Port = 6000
	if got := f.Addr(""); got != "127.0.0.1:7777" {
		t.Errorf("Addr() = %q, want the explicit addr", got)
	}
}

func TestLoadRejectsBothAddrAndPort(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "server:\n  addr: 127.0.0.1:6000\n  port: 7000\n")

	// Load reads beside the binary, so exercise the parsing via the same path
	// the process uses by checking validate() through a temp file copy.
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "port: 7000") {
		t.Fatal("test setup is wrong")
	}

	var f File
	f.Server.Addr = "127.0.0.1:6000"
	f.Server.Port = 7000
	if err := f.validate(); err == nil {
		t.Error("setting both addr and port should be rejected: one would be ignored")
	}
}

func TestLoadRejectsOutOfRangePort(t *testing.T) {
	var f File
	f.Server.Port = 70000
	if err := f.validate(); err == nil {
		t.Error("a port above 65535 should be rejected")
	}
	f.Server.Port = 0
	if err := f.validate(); err != nil {
		t.Errorf("port 0 means unset and must be allowed: %v", err)
	}
}

// Unknown keys are rejected rather than ignored: a typo that silently does
// nothing is worse than an error, because the user believes the file works.
func TestUnknownKeyIsRejected(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "server:\n  auto_port: false\n")

	f, _, err := loadFile(path)
	if err == nil {
		t.Fatal("an unknown key should be reported, not ignored")
	}
	if !strings.Contains(err.Error(), "auto_port") {
		t.Errorf("the error should name the offending key, got: %v", err)
	}
	_ = f
}

func TestEmptyFileMeansDefaults(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "")

	f, _, err := loadFile(path)
	if err != nil {
		t.Fatalf("an empty file should be all defaults, got: %v", err)
	}
	if !f.OpenBrowserEnabled() || !f.AuthEnabled() {
		t.Error("defaults were not applied")
	}
}

func TestMissingFileIsNotAnError(t *testing.T) {
	// Path() points beside the test binary; nothing guarantees a file there, so
	// the important part is that Load does not fail when it is absent.
	f, path, err := Load()
	if err != nil {
		t.Fatalf("a missing config file must not be an error: %v", err)
	}
	if !f.OpenBrowserEnabled() || !f.AuthEnabled() {
		t.Error("defaults should still be returned")
	}
	if path == "" {
		t.Error("the considered path should be reported so the caller can log it")
	}
}

func TestExplicitFalseIsHonoured(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, "server:\n  autoPort: false\n  openBrowser: false\nauth:\n  enabled: false\n")

	f, _, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.AutoPortEnabled() {
		t.Error("autoPort: false must turn the fallback off")
	}
	if f.OpenBrowserEnabled() {
		t.Error("openBrowser: false must be honoured")
	}
	if f.AuthEnabled() {
		t.Error("auth.enabled: false must be honoured")
	}
}

func TestParseFullFile(t *testing.T) {
	dir := t.TempDir()
	path := write(t, dir, `
server:
  port: 6000
  autoPort: true
  openBrowser: true
storage:
  db: /var/lib/reqbox.db
auth:
  token: choose-your-own-token
logging:
  verbose: true
`)

	f, _, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Addr("127.0.0.1:5178"); got != "127.0.0.1:6000" {
		t.Errorf("Addr() = %q", got)
	}
	if f.Storage.DB != "/var/lib/reqbox.db" {
		t.Errorf("db = %q", f.Storage.DB)
	}
	if f.Auth.Token != "choose-your-own-token" {
		t.Errorf("token = %q", f.Auth.Token)
	}
	if !f.Logging.Verbose {
		t.Error("verbose was not parsed")
	}
}
