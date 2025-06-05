#!/bin/bash

# Build script for AWS Lambda deployment
echo "Building Go Lambda function for AWS..."

# Set environment variables for Linux compilation
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# Build the binary
go build -ldflags="-s -w" -o bootstrap main.go

# Create deployment package
if [ -f bootstrap ]; then
    echo "Build successful! Creating deployment package..."
    zip lambda-deployment.zip bootstrap
    echo "Deployment package created: lambda-deployment.zip"
    
    # Clean up
    rm bootstrap
    echo "Build artifacts cleaned up."
else
    echo "Build failed!"
    exit 1
fi

echo "Ready for Lambda deployment!" 