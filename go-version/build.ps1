# PowerShell build script for AWS Lambda deployment
Write-Host "Building Go Lambda function for AWS..." -ForegroundColor Green

# Set environment variables for Linux compilation
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# Build the binary
Write-Host "Compiling binary..." -ForegroundColor Yellow
go build -ldflags="-s -w" -o bootstrap main.go

# Check if build was successful
if (Test-Path "bootstrap") {
    Write-Host "Build successful! Creating deployment package..." -ForegroundColor Green
    
    # Create deployment zip
    if (Test-Path "lambda-deployment.zip") {
        Remove-Item "lambda-deployment.zip"
    }
    
    Compress-Archive -Path "bootstrap" -DestinationPath "lambda-deployment.zip"
    Write-Host "Deployment package created: lambda-deployment.zip" -ForegroundColor Green
    
    # Clean up
    Remove-Item "bootstrap"
    Write-Host "Build artifacts cleaned up." -ForegroundColor Green
    
    Write-Host "Ready for Lambda deployment!" -ForegroundColor Cyan
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
} 