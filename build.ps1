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
go build -ldflags "-X main.version=$tag" -o gh-prx.exe .

Write-Host "`n✅ Ready — run: gh prx list"
