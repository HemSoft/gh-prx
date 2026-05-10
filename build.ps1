# Build, test, and install gh-prx
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Write-Host '--- vet ---'
go vet ./...

Write-Host '--- test ---'
go test ./...

Write-Host '--- build ---'
go build -o gh-prx.exe .

Write-Host "`n✅ Ready — run: gh prx list"
