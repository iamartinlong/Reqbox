package main

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// isAddrInUse reports whether err means "something is already listening here".
//
// Two mechanisms, because neither works everywhere:
//
//  1. errors.Is against syscall.EADDRINUSE. On Unix this is exact, and it is
//     what the standard library's net package returns.
//  2. A message match. On Windows the error is a *net.OpError wrapping
//     WSAEADDRINUSE, whose errno value is 10048 rather than EADDRINUSE, so the
//     sentinel comparison alone would never fire — and Windows is the platform
//     this feature exists for.
//
// Matching text is fragile, so it is the second resort and only ever tightens
// behaviour: if a future Go version rewords the message, the worst case is that
// the fallback stops happening and the clear bind error is reported instead.
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		msg := strings.ToLower(opErr.Err.Error())
		return strings.Contains(msg, "address already in use") ||
			strings.Contains(msg, "only one usage of each socket address") ||
			strings.Contains(msg, "通常每个套接字地址")
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}
