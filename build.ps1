# Build, test, and install gh-prx
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$tag = & git describe --tags --always 2>$null
if (-not $tag) { $tag = 'dev' }

Write-Host '--- vet ---'
go vet ./...

Write-Host '--- test ---'
go test ./...

Write-Host '--- build ---'
$date = Get-Date -Format 'yyyy-MM-dd'
go build -ldflags "-X main.version=$tag -X main.buildDate=$date" -o gh-prx.exe .

# Install into gh extension directory so `gh prx` uses the local build
$extDir = Join-Path $env:LOCALAPPDATA 'GitHub CLI\extensions\gh-prx'
if (Test-Path $extDir) {
    Copy-Item gh-prx.exe (Join-Path $extDir 'gh-prx.exe') -Force
    Write-Host "`n✅ Built & installed ($tag) — run: gh prx"
} else {
    Write-Host "`n✅ Built ($tag) — extension dir not found, run: gh extension install ."
}
