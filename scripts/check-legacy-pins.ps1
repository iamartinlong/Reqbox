# Verify the Go 1.20 compatibility pins have not drifted.
#
# Why this exists: on this branch, four dependencies must stay at versions that
# still compile with Go 1.20. `go mod tidy` (and `go get` on any related module)
# will happily raise them to releases that require Go 1.23+, at which point the
# branch no longer builds and the failure looks unrelated to the cause.
#
#   .\scripts\check-legacy-pins.ps1
#
# Exit code 0 = all pins intact, 1 = something drifted.

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$goMod = Join-Path $root "go.mod"

if (-not (Test-Path $goMod)) { throw "go.mod not found at $goMod" }

# Pins required by the Go 1.20 build. See docs/LEGACY-WINDOWS.md.
$required = [ordered]@{
    "go "                = "1.20"
    "modernc.org/sqlite" = "v1.29.10"
    "modernc.org/libc"   = "v1.55.3"
    "modernc.org/memory" = "v1.8.0"
    "golang.org/x/sys"   = "v0.22.0"
}

$lines = Get-Content $goMod
$failed = @()

foreach ($module in $required.Keys) {
    $expected = $required[$module]

    if ($module -eq "go ") {
        $line = $lines | Where-Object { $_ -match '^go\s+' } | Select-Object -First 1
        $actual = if ($line) { ($line -replace '^go\s+', '').Trim() } else { "" }
    }
    else {
        $line = $lines | Where-Object { $_ -match "^\s*$([regex]::Escape($module))\s+" } | Select-Object -First 1
        $actual = if ($line) { (($line.Trim() -split '\s+')[1]) } else { "" }
    }

    if ($actual -eq $expected) {
        Write-Host ("  OK      {0,-22} {1}" -f $module.Trim(), $actual)
    }
    else {
        Write-Host ("  DRIFTED {0,-22} expected {1}, found '{2}'" -f $module.Trim(), $expected, $actual)
        $failed += "$module (expected $expected, found '$actual')"
    }
}

Write-Host ""

if ($failed.Count -gt 0) {
    Write-Host "Compatibility pins have drifted. This branch will not build with Go 1.20." -ForegroundColor Red
    Write-Host ""
    Write-Host "Fix with explicit requires, then re-run:"
    foreach ($module in $required.Keys) {
        if ($module -eq "go ") { continue }
        Write-Host ("  go mod edit -require={0}@{1}" -f $module, $required[$module])
    }
    Write-Host "  go mod edit -go=1.20"
    Write-Host ""
    Write-Host "Do not run plain 'go mod tidy' afterwards; it raises them again."
    Write-Host "See docs/LEGACY-WINDOWS.md for why these versions are pinned."
    exit 1
}

Write-Host "All Go 1.20 compatibility pins intact." -ForegroundColor Green
exit 0
