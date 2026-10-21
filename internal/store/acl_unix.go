//go:build !windows

package store

import (
	"fmt"
	"os"
)

// restrictToCurrentUser narrows the file to its owner.
//
// On Unix the mode bits are the whole story, so os.Chmod is enough. Modes are
// also the only place that can protect the -wal and -shm sidecars, which hold
// the same data; they are created by SQLite, so the directory mode is what
// constrains them.
func restrictToCurrentUser(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict %s: %w", path, err)
	}
	return nil
}

// restrictDir tightens the data directory to its owner.
func restrictDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return os.Chmod(path, 0o700)
}
