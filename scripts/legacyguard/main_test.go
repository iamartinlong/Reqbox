package main

import (
	"strings"
	"testing"
)

// The whole point of this command is to fail when a newer toolchain sneaks in.
// A test that only checks the happy path would pass against a binary that
// cannot start, so both directions are covered.
func TestCheckForbiddenRejectsProcessPrng(t *testing.T) {
	cases := []string{
		"ProcessPrng",
		"call ProcessPrng at runtime",
		"bcryptprimitives.dll",
	}

	for _, raw := range cases {
		if err := checkForbidden([]byte(raw)); err == nil {
			t.Errorf("checkForbidden(%q) = nil, want an error", raw)
		}
	}
}

func TestCheckForbiddenAllowsLegacyImports(t *testing.T) {
	// A Go 1.20 binary's import table is exactly this shape.
	raw := build("kernel32.dll\x00AddVectoredExceptionHandler\x00CreateFileA\x00")

	if err := checkForbidden(raw); err != nil {
		t.Errorf("checkForbidden rejected a clean binary: %v", err)
	}
}

// The error has to say what to do, because the failure appears on a machine the
// developer does not have access to.
func TestCheckForbiddenErrorIsActionable(t *testing.T) {
	err := checkForbidden([]byte("ProcessPrng"))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"Go 1.20", "2008 R2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}

func build(parts ...string) []byte {
	return []byte(strings.Join(parts, ""))
}
