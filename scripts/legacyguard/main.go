// Command legacyguard checks that a Windows binary can actually start on
// Windows Server 2008 R2 (NT 6.1) and older.
//
// Why this exists: a wrong toolchain produces a binary that cross-compiles
// cleanly, passes `go test`, and then exits immediately on the target machine
// without printing anything. The PE header alone does not catch it — Go 1.21+
// still stamps 6.1 as the minimum OS while resolving ProcessPrng through
// LoadLibrary at startup, so the import table looks innocent.
//
// Three independent checks, each covering a different failure mode:
//
//  1. PE minimum OS is 6.x (the linker's own statement about compatibility).
//  2. No reference to the APIs that Go 1.21+ hard-requires (ProcessPrng) or to
//     the DLL that provides it. Catches an accidental newer toolchain even if
//     the PE header was overridden.
//  3. Every statically imported DLL is on an allowlist of libraries that exist
//     on NT 6.1. A stray import of bcryptprimitives.dll, api-ms-win-*, or a
//     UCRT-only dependency would fail here rather than on the customer's box.
//
// Usage:
//
//	go run ./scripts/legacyguard -exe dist/reqbox-win-legacy-amd64.exe
//
// Exit code 0 = safe to ship to legacy Windows. 1 = do not ship.
package main

import (
	"bytes"
	"debug/pe"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// legacyDLLs are libraries present on a stock Windows Server 2008 R2 (NT 6.1).
// Kept deliberately short: if a new import shows up, the build fails and
// someone has to verify that library exists on 6.1 before adding it here.
//
// What must not appear:
//   - bcryptprimitives.dll — exists on 6.1, but ProcessPrng (the symbol Go 1.21+
//     calls) does not. Loading the DLL is not the problem; the missing export is.
//   - ucrtbase.dll / api-ms-win-crt-* — the Universal CRT is not installed by
//     default on 2008 R2.
var legacyDLLs = map[string]bool{
	"kernel32.dll": true,
	"ntdll.dll":    true,
	"user32.dll":   true,
	"advapi32.dll": true,
	"ws2_32.dll":   true,
	"mswsock.dll":  true,
	"secur32.dll":  true,
	"crypt32.dll":  true,
	"iphlpapi.dll": true,
	"dnsapi.dll":   true,
	"ole32.dll":    true,
	"shell32.dll":  true,
	"psapi.dll":    true,
	"userenv.dll":  true,
	"netapi32.dll": true,
}

// forbiddenStrings are substrings that must not appear anywhere in the file.
// These are resolved dynamically, so an import-table check alone misses them.
var forbiddenStrings = []string{
	"ProcessPrng",
	"bcryptprimitives",
	"bcryptprimitives.dll",
}

func main() {
	exe := flag.String("exe", "", "path to the Windows executable to check")
	flag.Parse()

	if *exe == "" {
		fmt.Fprintln(os.Stderr, "legacyguard: -exe is required")
		os.Exit(2)
	}
	if err := run(*exe); err != nil {
		fmt.Fprintf(os.Stderr, "legacyguard: FAIL\n  %v\n", err)
		os.Exit(1)
	}
	fmt.Println("legacyguard: OK — safe for Windows Server 2008 R2 and later 6.x")
}

func run(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	f, err := pe.Open(path)
	if err != nil {
		return fmt.Errorf("not a PE file: %w", err)
	}
	defer f.Close()

	// 1. Header check.
	var major, minor uint16
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		major, minor = oh.MajorOperatingSystemVersion, oh.MinorOperatingSystemVersion
	case *pe.OptionalHeader32:
		major, minor = oh.MajorOperatingSystemVersion, oh.MinorOperatingSystemVersion
	default:
		return fmt.Errorf("unknown optional header type %T", f.OptionalHeader)
	}
	if major != 6 {
		return fmt.Errorf(
			"PE minimum OS is %d.%d, expected 6.x (NT 6.1 = Server 2008 R2). "+
				"A 10.0 here means the binary was produced by a toolchain that "+
				"refuses to load on legacy Windows", major, minor)
	}
	fmt.Printf("  PE minimum OS: %d.%d OK\n", major, minor)

	// 2. Dynamically resolved APIs.
	if err := checkForbidden(raw); err != nil {
		return err
	}
	fmt.Println("  forbidden APIs: absent OK")

	// 3. Static imports.
	symbols, err := f.ImportedSymbols()
	if err != nil {
		return fmt.Errorf("read imports: %w", err)
	}
	seen := map[string]bool{}
	var unknown []string
	for _, sym := range symbols {
		// ImportedSymbols returns "func:library.dll".
		i := strings.LastIndex(sym, ":")
		if i < 0 {
			return fmt.Errorf("unexpected import format %q", sym)
		}
		dll := strings.ToLower(sym[i+1:])
		if seen[dll] {
			continue
		}
		seen[dll] = true
		if !legacyDLLs[dll] {
			unknown = append(unknown, dll)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf(
			"imports DLLs not verified to exist on NT 6.1: %s\n"+
				"  Verify the library ships with Windows Server 2008 R2, then add "+
				"it to legacyDLLs in this file", strings.Join(unknown, ", "))
	}

	libs := make([]string, 0, len(seen))
	for dll := range seen {
		libs = append(libs, dll)
	}
	sort.Strings(libs)
	fmt.Printf("  imported DLLs (%d): %s OK\n", len(libs), strings.Join(libs, ", "))

	if n := len(symbols); n > 0 {
		fmt.Printf("  imported functions: %d\n", n)
	}
	return nil
}

// checkForbidden looks for the APIs that Go 1.21+ resolves at startup. Split
// out from run so it can be tested without crafting a full PE file.
func checkForbidden(raw []byte) error {
	for _, needle := range forbiddenStrings {
		if bytes.Contains(raw, []byte(needle)) {
			return fmt.Errorf(
				"binary references %q. Go 1.21+ requires ProcessPrng, which "+
					"bcryptprimitives.dll does not export on Server 2008 R2; the "+
					"process would exit before printing anything. Rebuild with Go 1.20",
				needle)
		}
	}
	return nil
}
