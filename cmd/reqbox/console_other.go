//go:build !windows

package main

// allowConsoleCopy is a no-op away from Windows: there is no console mode with
// a QuickEdit flag, and terminal emulators already copy the way people expect.
func allowConsoleCopy() {}
