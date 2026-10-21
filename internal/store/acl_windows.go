//go:build windows

package store

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// restrictToCurrentUser replaces the file's DACL with one granting access only
// to the current user, SYSTEM and Administrators.
//
// os.Chmod cannot do this: on Windows it only toggles the read-only attribute.
// A file created with 0600 therefore keeps whatever ACL it inherits from its
// parent directory, which for a normal install means every local user can read
// the database and the credentials inside it.
func restrictToCurrentUser(path string) error {
	return setACL(path, false)
}

// restrictDir tightens the data directory so files created beside the database
// inherit a restrictive DACL. The -wal and -shm sidecars hold the same data, and
// SQLite creates them itself, so the directory is the only place that can
// constrain them.
func restrictDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return setACL(path, true)
}

// setACL installs the restrictive DACL. inheritable is true for directories,
// where the ACEs must propagate to files SQLite creates later.
func setACL(path string, inheritable bool) error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return fmt.Errorf("open process token: %w", err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("token user: %w", err)
	}

	systemSID, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return fmt.Errorf("system sid: %w", err)
	}
	adminsSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return fmt.Errorf("administrators sid: %w", err)
	}

	build := allowEntry
	if inheritable {
		build = allowEntryInheritable
	}
	entries := []windows.EXPLICIT_ACCESS{
		build(user.User.Sid, windows.GENERIC_ALL),
		build(systemSID, windows.GENERIC_ALL),
		build(adminsSID, windows.GENERIC_ALL),
	}

	// A nil base ACL means "start from nothing", so the result contains exactly
	// these entries and no inherited ones.
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("build acl: %w", err)
	}

	// PROTECTED_DACL_SECURITY_INFORMATION stops the parent directory's
	// inheritable entries from being applied again on the next write, which
	// would silently undo this.
	err = windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil,
	)
	if err != nil {
		return fmt.Errorf("set dacl on %s: %w", path, err)
	}
	return nil
}

func allowEntry(sid *windows.SID, mask windows.ACCESS_MASK) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: mask,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

// allowEntryInheritable is allowEntry but marks the ACE to propagate to files
// created inside a directory.
//
// This matters for SQLite: it creates the -wal and -shm sidecars itself, and
// they hold the same rows as the database. Without an inheritable ACE those
// files would only get the permissions their own creation call requests, which
// on Windows includes the per-logon-session SID and no Administrators entry.
func allowEntryInheritable(sid *windows.SID, mask windows.ACCESS_MASK) windows.EXPLICIT_ACCESS {
	e := allowEntry(sid, mask)
	e.Inheritance = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	return e
}
