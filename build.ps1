$ErrorActionPreference = "Stop"

Push-Location $PSScriptRoot
try {
    go -C src vet ./...
    if ($LASTEXITCODE -ne 0) { exit 1 }

    go -C src test ./...
    if ($LASTEXITCODE -ne 0) { exit 1 }

    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go -C src build -trimpath -ldflags "-s -w" -o ../cdifnos/app/server/cdi-server .
    if ($LASTEXITCODE -ne 0) { exit 1 }

    & "$PSScriptRoot\tools\fnpack.exe" build -d cdifnos
    if ($LASTEXITCODE -ne 0) { exit 1 }
} finally {
    Pop-Location
}
