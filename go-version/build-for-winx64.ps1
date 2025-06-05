# PowerShell build script for Windows x64 Go binary
Write-Host "Building Go binary for Windows x64..." -ForegroundColor Green

# Set environment variables for Windows build
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# Remove any previous build artifacts
if (Test-Path "windows-scraper.exe") {
    Remove-Item "windows-scraper.exe" -Force
}

# Build the Go binary for Windows
Write-Host "Compiling Go binary..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o "windows-scraper.exe" "main.go"

# Verify build success
if (Test-Path "windows-scraper.exe") {
    Write-Host "Build successful! Executable created: windows-scraper.exe" -ForegroundColor Green
    Write-Host "To run: .\windows-scraper.exe" -ForegroundColor Cyan
    Write-Host "-------------------------------------------"
    Write-Host "To install locally, you can:"
    Write-Host "New-Item -ItemType Directory -Path `"$env:USERPROFILE\Desktop`""
    Write-Host "Copy-Item -Path `"windows-scraper.exe`" -Destination `"$env:USERPROFILE\Desktop`""
    Write-Host "Double-click the executable or run from the command line."
    Write-Host "-------------------------------------------"
} else {
    Write-Host "Build failed! windows-scraper.exe was not created." -ForegroundColor Red
    exit 1
}