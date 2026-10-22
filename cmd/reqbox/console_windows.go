//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// allowConsoleCopy makes the console window cooperate with copying.
//
// Why: a double-clicked reqbox.exe gets a console window the user did not ask
// for, and the first thing they try to do is drag-select the printed token URL.
// With QuickEdit enabled (the Windows default) the selection pauses the process
// and Ctrl+C copies the selection instead of interrupting — so the obvious
// gestures fight each other and the window looks frozen.
//
// Turning QuickEdit off fixes the "frozen" half: the program keeps printing, and
// Ctrl+C means Ctrl+C. It does not make the text selectable-with-the-mouse in the
// old way, but the browser is opened automatically now, which is the path we
// actually want people to use.
//
// Everything here is best effort. When stdout is redirected to a file or a pipe
// (CI, a service wrapper) there is no console mode to change, and that is fine.
func allowConsoleCopy() {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return
	}
	// QuickEdit is the flag that turns a click into a selection and freezes
	// output. Clearing it also requires EXTENDED_FLAGS to be set, or Windows
	// ignores the change.
	mode &^= windows.ENABLE_QUICK_EDIT_MODE
	mode |= windows.ENABLE_EXTENDED_FLAGS
	_ = windows.SetConsoleMode(h, mode)
}
