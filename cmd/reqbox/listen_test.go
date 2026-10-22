package main

import (
	"net"
	"strconv"
	"testing"
)

// occupy binds a port and returns it, so the test can be sure the fallback is
// exercising a real conflict rather than a simulated error.
func occupy(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln, ln.Addr().(*net.TCPAddr).Port
}

func TestListenWithFallbackMovesToNextPort(t *testing.T) {
	_, busy := occupy(t)
	addr := "127.0.0.1:" + strconv.Itoa(busy)

	ln, actual, err := listenWithFallback(addr, true)
	if err != nil {
		t.Fatalf("expected the server to move to a free port, got: %v", err)
	}
	defer ln.Close()

	if actual == addr {
		t.Fatalf("reported %q as both requested and actual; it was busy", addr)
	}
	want := "127.0.0.1:" + strconv.Itoa(busy+1)
	if actual != want {
		t.Errorf("actual = %q, want the next port %q", actual, want)
	}

	// The returned listener has to be the one that moved, or the reported port
	// would be a lie.
	if got := ln.Addr().(*net.TCPAddr).Port; strconv.Itoa(got) != strconv.Itoa(busy+1) {
		t.Errorf("listener is on port %d, but %q was reported", got, actual)
	}
}

// With the fallback disabled the original error has to survive, because the
// caller prints it and the operator needs to recognise a port conflict.
func TestListenWithFallbackDisabled(t *testing.T) {
	_, busy := occupy(t)
	addr := "127.0.0.1:" + strconv.Itoa(busy)

	ln, actual, err := listenWithFallback(addr, false)
	if err == nil {
		if ln != nil {
			ln.Close()
		}
		t.Fatal("expected an error when the port is busy and fallback is off")
	}
	if actual != addr {
		t.Errorf("actual = %q, want the requested address", actual)
	}
	if !isAddrInUse(err) {
		t.Errorf("the error should be recognisable as 'in use', got: %v", err)
	}
}

func TestListenWithFallbackFreePort(t *testing.T) {
	// Ask for any free port: the first attempt must succeed and be reported
	// as-is, since walking a port range is only for conflicts.
	ln, actual, err := listenWithFallback("127.0.0.1:0", true)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if actual != "127.0.0.1:0" {
		t.Errorf("actual = %q; port 0 means 'any', so it should be reported unchanged", actual)
	}
	// The listener still knows the real port, which is what the startup banner
	// now uses.
	if ln.Addr().(*net.TCPAddr).Port == 0 {
		t.Error("the listener should expose a concrete port")
	}
}

// Port 0 is not a conflict, so the fallback must not walk away from it.
func TestListenWithFallbackDoesNotWalkFromZero(t *testing.T) {
	ln, actual, err := listenWithFallback("127.0.0.1:0", true)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if actual != "127.0.0.1:0" {
		t.Errorf("actual = %q, want the requested :0", actual)
	}
}

func TestIsAddrInUseIgnoresUnrelatedErrors(t *testing.T) {
	if isAddrInUse(nil) {
		t.Error("nil is not a port conflict")
	}
	if isAddrInUse(&net.OpError{Op: "listen", Err: errString("permission denied")}) {
		t.Error("permission denied is not a port conflict")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
