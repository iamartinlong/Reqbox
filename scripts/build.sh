#!/usr/bin/env sh
# Build ApiMock end to end: web client, then the Go binary that embeds it.
#
#   ./scripts/build.sh                 # current platform
#   ./scripts/build.sh 0.2.0           # with an explicit version
#   ./scripts/build.sh 0.2.0 --all     # linux/darwin/windows matrix
#   ./scripts/build.sh dev --skip-web  # reuse an existing web/dist

set -eu

VERSION="${1:-dev}"
ALL=""
SKIP_WEB=""
for arg in "$@"; do
  case "$arg" in
    --all) ALL="1" ;;
    --skip-web) SKIP_WEB="1" ;;
  esac
done

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST="$ROOT/dist"
mkdir -p "$DIST"

if [ -z "$SKIP_WEB" ]; then
  echo "==> building web client"
  (cd "$ROOT/web" && pnpm install --frozen-lockfile && pnpm build)
fi

if [ ! -f "$ROOT/web/dist/index.html" ]; then
  echo "web/dist/index.html is missing. Run without --skip-web first." >&2
  exit 1
fi

LDFLAGS="-s -w -X apimock/internal/server.Version=$VERSION"
export CGO_ENABLED=0

build() {
  os="$1"; arch="$2"; out="$3"
  echo "==> building $out"
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/$out" ./cmd/apimock
}

cd "$ROOT"
if [ -n "$ALL" ]; then
  build linux   amd64 apimock-linux-amd64
  build linux   arm64 apimock-linux-arm64
  build darwin  amd64 apimock-darwin-amd64
  build darwin  arm64 apimock-darwin-arm64
  build windows amd64 apimock-windows-amd64.exe
else
  build "$(go env GOOS)" "$(go env GOARCH)" apimock
fi

echo "==> done. Artifacts in $DIST"
