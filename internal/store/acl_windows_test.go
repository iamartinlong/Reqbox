//go:build windows

package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// broadPrincipals are the groups a file inherits from an ordinary directory.
// Any of them in the DACL means every local account can read the credentials.
var broadPrincipals = map[string]string{
	"S-1-1-0":      "Everyone",
	"S-1-5-11":     "Authenticated Users",
	"S-1-5-32-545": "BUILTIN\\Users",
}

// assertNotBroad fails if any given file grants access to a broad principal.
func assertNotBroad(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		full := filepath.Join(dir, file)
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("%s should exist: %v", file, err)
		}
		acl, err := windows.GetNamedSecurityInfo(
			full, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatalf("%s: read acl: %v", file, err)
		}
		dacl, _, err := acl.DACL()
		if err != nil || dacl == nil {
			t.Fatalf("%s: no dacl", file)
		}
		for i := uint32(0); i < uint32(dacl.AceCount); i++ {
			var ace *windows.ACCESS_ALLOWED_ACE
			if err := windows.GetAce(dacl, i, &ace); err != nil {
				t.Fatalf("%s: ace %d: %v", file, i, err)
			}
			if ace == nil {
				continue
			}
			// The SID is a variable-length field starting at SidStart.
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
			for prefix, label := range broadPrincipals {
				if strings.Contains(sid, prefix) {
					t.Errorf("%s: DACL grants access to %s", file, label)
				}
			}
		}
	}
}

// The database holds plaintext credentials for every saved request, so a
// default-permissive ACL exposes them to every local account. The failure is
// silent - the app works fine and nobody notices - which is why it is tested.
//
// The directory is created explicitly with a permissive ACL first. t.TempDir()
// already sits under a restricted parent, so using it would let the check pass
// even if the code did nothing.
func TestDatabaseIsNotWorldReadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Widen it, so any pass can only come from our own restriction.
	if err := grantUsers(dir); err != nil {
		t.Fatalf("could not set up a permissive directory: %v", err)
	}

	s, err := Open(filepath.Join(dir, "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.PermissionWarning != nil {
		t.Fatalf("restricting permissions failed: %v", s.PermissionWarning)
	}

	// Force the sidecars to exist: they carry the same rows, so leaving them
	// world-readable would defeat the point.
	if _, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES ('probe','x')`); err != nil {
		t.Fatal(err)
	}

	assertNotBroad(t, dir, []string{"api.db", "api.db-wal", "api.db-shm"})
}

// grantUsers adds an inheritable grant for BUILTIN\Users, reproducing what a
// file gets in a normal directory.
func grantUsers(path string) error {
	usersSID, err := windows.CreateWellKnownSid(windows.WinBuiltinUsersSid)
	if err != nil {
		return err
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
			TrusteeValue: windows.TrusteeValueFromSID(usersSID),
		},
	}}, nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil,
	)
}
