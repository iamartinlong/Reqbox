# Build ApiMock end to end: web client, then the Go binary that embeds it.
#
#   .\scripts\build.ps1                      # current platform
#   .\scripts\build.ps1 -Version 0.2.0       # with an explicit version
#   .\scripts\build.ps1 -All                 # linux/darwin/windows matrix
#   .\scripts\build.ps1 -SkipWeb             # reuse an existing web/dist

param(
    [string]$Version = "dev",
    [switch]$All,
    [switch]$SkipWeb
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

if (-not $SkipWeb) {
    Write-Host "==> building web client"
    Push-Location (Join-Path $root "web")
    try {
        pnpm install --frozen-lockfile
        pnpm build
    }
    finally {
        Pop-Location
    }
}

if (-not (Test-Path (Join-Path $root "web/dist/index.html"))) {
    throw "web/dist/index.html is missing. Run without -SkipWeb first."
}

$ldflags = "-s -w -X apimock/internal/server.Version=$Version"

if ($All) {
    $targets = @(
        @{ OS = "linux";   Arch = "amd64"; Out = "apimock-linux-amd64" },
        @{ OS = "linux";   Arch = "arm64"; Out = "apimock-linux-arm64" },
        @{ OS = "darwin";  Arch = "amd64"; Out = "apimock-darwin-amd64" },
        @{ OS = "darwin";  Arch = "arm64"; Out = "apimock-darwin-arm64" },
        @{ OS = "windows"; Arch = "amd64"; Out = "apimock-windows-amd64.exe" }
    )
    foreach ($t in $targets) {
        Write-Host "==> building $($t.Out)"
        $env:GOOS = $t.OS
        $env:GOARCH = $t.Arch
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags $ldflags -o (Join-Path $dist $t.Out) ./cmd/apimock
        if ($LASTEXITCODE -ne 0) { throw "build failed for $($t.Out)" }
    }
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
else {
    $out = if ($IsWindows) { "apimock.exe" } else { "apimock" }
    Write-Host "==> building $out"
    Push-Location $root
    try {
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags $ldflags -o (Join-Path $dist $out) ./cmd/apimock
        if ($LASTEXITCODE -ne 0) { throw "build failed" }
    }
    finally {
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Pop-Location
    }
}

Write-Host "==> done. Artifacts in $dist"
