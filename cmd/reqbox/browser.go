package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser launches the default browser at url, best effort.
//
// Why this exists: a double-clicked binary gets a console window the user did
// not ask for and cannot easily copy from, while the one thing they need — a
// sign-in link carrying the token — scrolls past in it. Opening the browser
// removes the copy step entirely.
//
// Failure is never fatal. There is no browser on a server, no handler
// registered on some desktops, and no DISPLAY over SSH; in all of those cases
// the printed URL is still there and still correct. The caller logs the failure
// at debug level and carries on.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// rundll32 with FileProtocolHandler uses the shell's URL handler, which
		// is the documented way to do this from a process with no console
		// attached. `cmd /c start` would split on '&' in the URL.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browser: %w", err)
	}
	// Reap the child; nobody is waiting on its exit status. Detaching matters on
	// Windows, where a failed launch would otherwise leave a zombie cmd window.
	go func() { _ = cmd.Wait() }()
	return nil
}
