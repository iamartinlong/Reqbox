#!/usr/bin/env sh
# Build the legacy Windows binary (Go 1.20 toolchain).
#
# This branch must be built with Go 1.20: it is the last release that runs on
# Windows Server 2016 之前的系统. A newer toolchain compiles fine but produces a
# binary that exits immediately on those machines, so this script refuses to run
# with the wrong version rather than producing a silent trap.
#
#   ./scripts/build-legacy.sh
#   ./scripts/build-legacy.sh 0.1.0
#   GOROOT_120=/usr/local/go1.20 ./scripts/build-legacy.sh --skip-web
#
# Go 1.20.14: https://go.dev/dl/

set -eu

VERSION="${1:-dev}"
SKIP_WEB=""
for arg in "$@"; do
  case "$arg" in
    --skip-web) SKIP_WEB="1" ;;
  esac
done

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST="$ROOT/dist"
mkdir -p "$DIST"

# --- locate a Go 1.20 toolchain ---------------------------------------------
find_go120() {
  if [ -n "${GOROOT_120:-}" ] && [ -x "$GOROOT_120/bin/go" ]; then
    printf '%s\n' "$GOROOT_120/bin/go"
    return 0
  fi
  for candidate in /usr/local/go1.20 /usr/lib/go-1.20 "$HOME/sdk/go1.20.14"; do
    if [ -x "$candidate/bin/go" ]; then
      printf '%s\n' "$candidate/bin/go"
      return 0
    fi
  done
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  return 1
}

GO_EXE=$(find_go120) || {
  echo "No Go toolchain found. Install Go 1.20.14 (https://go.dev/dl/) or set GOROOT_120." >&2
  exit 1
}

GO_VERSION=$("$GO_EXE" version)
echo "==> toolchain: $GO_VERSION"

case "$GO_VERSION" in
  *go1.20|*go1.20.*) ;;
  *)
    cat >&2 <<EOF
This branch must be built with Go 1.20, but found:
  $GO_VERSION

Reason: Go 1.21+ emits binaries that call ProcessPrng and cannot start on
Windows Server 2016 and earlier. The build would succeed and then fail on the
target machine with no error message.

Install Go 1.20.14 and set GOROOT_120, or put it earlier in PATH.
EOF
    exit 1
    ;;
esac

# Stops a newer go command from swapping itself for a newer toolchain after
# reading the `go 1.20` directive in go.mod.
export GOTOOLCHAIN=local

# --- web client --------------------------------------------------------------
if [ -z "$SKIP_WEB" ]; then
  echo "==> building web client"
  (cd "$ROOT/web" && pnpm install --frozen-lockfile && pnpm build)
fi

if [ ! -f "$ROOT/web/dist/index.html" ]; then
  echo "web/dist/index.html is missing. Run without --skip-web first." >&2
  exit 1
fi

# --- binary ------------------------------------------------------------------
LDFLAGS="-s -w -X reqbox/internal/server.Version=$VERSION"
OUT="$DIST/reqbox-win-legacy-amd64.exe"

echo "==> building $OUT"
cd "$ROOT"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "$GO_EXE" build -trimpath -ldflags "$LDFLAGS" -o "$OUT" ./cmd/reqbox

# --- verify ------------------------------------------------------------------
# A wrong toolchain still produces a working-looking binary, so check what
# actually determines whether it runs on the old systems. The PE header alone is
# not enough: Go 1.21+ stamps 6.1 while resolving ProcessPrng at startup, so the
# header looks fine and the process still dies on 2008 R2. `legacyguard` checks
# the header, the forbidden runtime APIs, and the imported DLLs.
cd "$ROOT"
"$GO_EXE" run ./scripts/legacyguard -exe "$OUT"

echo "==> built: $OUT"
echo "==> done"
