package main

import (
	"fmt"
	"net"
	"strconv"
)

// maxPortAttempts caps how far the fallback will walk before giving up. Walking
// thousands of ports would turn "the port is busy" into a hang, which is a worse
// answer than a clear error.
const maxPortAttempts = 20

// listenWithFallback binds addr, optionally moving to the next free port.
//
// The first port tried is always the one the operator asked for, so a stable
// address stays stable. Moving is opt-out (`server.autoPort: false`) and the
// caller logs it: silently serving somewhere other than what the config says is
// the kind of surprise that costs an hour of debugging later.
//
// Port 0 is not walked. An explicit 0 already means "let the kernel choose", so
// there is nothing to fall back from.
func listenWithFallback(addr string, allowFallback bool) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, addr, nil
	}
	if !allowFallback || !isAddrInUse(err) {
		return nil, addr, err
	}

	host, portStr, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		return nil, addr, err
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil || port == 0 {
		return nil, addr, err
	}

	for i := 1; i <= maxPortAttempts; i++ {
		candidate := net.JoinHostPort(host, strconv.Itoa(port+i))
		if ln, tryErr := net.Listen("tcp", candidate); tryErr == nil {
			return ln, candidate, nil
		}
	}
	return nil, addr, fmt.Errorf(
		"%w (also tried %d ports after %d; change server.port in reqbox.yaml)",
		err, maxPortAttempts, port)
}
