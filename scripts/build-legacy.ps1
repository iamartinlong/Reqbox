# Build the legacy Windows binary (Go 1.20 toolchain).
#
# This branch must be built with Go 1.20: it is the last release that runs on
# Windows Server 2016 之前的系统. A newer toolchain will succeed at compiling
# but produce a binary that exits immediately on those machines, so this script
# refuses to run with the wrong version instead of producing a silent trap.
#
#   .\scripts\build-legacy.ps1                      # windows/amd64, version dev
#   .\scripts\build-legacy.ps1 -Version 0.1.0
#   .\scripts\build-legacy.ps1 -GoRoot D:\go120     # explicit toolchain
#   .\scripts\build-legacy.ps1 -SkipWeb             # reuse web/dist
#
# The Go 1.20 toolchain can be obtained from https://go.dev/dl/ (go1.20.14).

param(
    [string]$Version = "dev",
    [string]$GoRoot = "",
    [switch]$SkipWeb
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

# --- locate a Go 1.20 toolchain ---------------------------------------------
function Resolve-Go120 {
    param([string]$Explicit)

    if ($Explicit) {
        # Accept either a GOROOT (contains bin\go.exe) or the bin directory itself.
        foreach ($exe in @(
            (Join-Path $Explicit "bin\go.exe"),
            (Join-Path $Explicit "go.exe")
        )) {
            if (Test-Path $exe) { return $exe }
        }
        throw "-GoRoot '$Explicit' does not contain bin\go.exe (nor go.exe directly)."
    }

    # Common install layouts: a bare GOROOT, or a parent folder holding one.
    $candidates = @(
        "D:\go120\go", "C:\go120\go",
        "D:\go120", "C:\go120",
        "$env:USERPROFILE\sdk\go1.20.14",
        "C:\Program Files\Go-1.20"
    )
    foreach ($candidate in $candidates) {
        $exe = Join-Path $candidate "bin\go.exe"
        if (Test-Path $exe) { return $exe }
    }

    $onPath = Get-Command go -ErrorAction SilentlyContinue
    if ($onPath) { return $onPath.Source }

    throw "No Go toolchain found. Download Go 1.20.14 from https://go.dev/dl/ and pass -GoRoot."
}

$goExe = Resolve-Go120 -Explicit $GoRoot

$goVersion = & $goExe version
Write-Host "==> toolchain: $goVersion"

if ($goVersion -notmatch 'go1\.20(\.|$)') {
    throw @"
This branch must be built with Go 1.20, but found:
  $goVersion

Reason: Go 1.21+ emits binaries that call ProcessPrng and cannot start on
Windows Server 2016 and earlier. The build would succeed and then fail on the
target machine with no error message.

Install Go 1.20.14 and pass -GoRoot, or put it earlier in PATH.
"@
}

# GOTOOLCHAIN=local stops a newer go command from silently swapping itself out
# for a newer toolchain after reading the `go 1.20` directive in go.mod.
$env:GOTOOLCHAIN = "local"

# --- web client --------------------------------------------------------------
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

# --- binary ------------------------------------------------------------------
$ldflags = "-s -w -X apimock/internal/server.Version=$Version"
$out = Join-Path $dist "apimock-win-legacy-amd64.exe"

Write-Host "==> building $out"
Push-Location $root
try {
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    & $goExe build -trimpath -ldflags $ldflags -o $out ./cmd/apimock
    if ($LASTEXITCODE -ne 0) { throw "build failed" }
}
finally {
    Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

# --- verify the result -------------------------------------------------------
# A wrong toolchain still produces a working-looking binary, so check the two
# things that actually determine whether it runs on the old systems.
$bytes = [IO.File]::ReadAllBytes($out)
$peOff = [BitConverter]::ToInt32($bytes, 0x3C)
$optOff = $peOff + 24
$osMajor = [BitConverter]::ToUInt16($bytes, $optOff + 40)
$osMinor = [BitConverter]::ToUInt16($bytes, $optOff + 42)

# PE min OS 6.1 = Windows 7 / Server 2008 R2. Go 1.27+ stamps 10.0 here.
if ($osMajor -ne 6) {
    throw "PE minimum OS is $osMajor.$osMinor, expected 6.x. The binary would not load on legacy Windows."
}

Write-Host "==> PE minimum OS: $osMajor.$osMinor (OK)"
Write-Host "==> built: $out"
Write-Host "    $(& $goExe version)"
Write-Host "==> done"
