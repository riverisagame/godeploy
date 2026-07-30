<#
.SYNOPSIS
Builds the UI and Docker image in a fail-fast pipeline.
#>

$ErrorActionPreference = "Stop"

Write-Host "Step 1: Building frontend (UI)..." -ForegroundColor Cyan
Set-Location "web"
try {
    # Check if npm is available
    if (Get-Command npm -ErrorAction SilentlyContinue) {
        Write-Host "Running npm install..."
        npm install
        
        Write-Host "Running npm run build..."
        npm run build
    } else {
        Write-Warning "npm not found. Skipping frontend build (Not Recommended). Please install Node.js."
    }
} finally {
    Set-Location ".."
}
Write-Host "Frontend build completed." -ForegroundColor Green

Write-Host "Step 2: Building Docker image..." -ForegroundColor Cyan
try {
    docker build -t godeploy:latest .
} catch {
    Write-Error "Docker build failed!"
    exit 1
}
Write-Host "Docker image build completed." -ForegroundColor Green

Write-Host "Full deployment pipeline completed successfully." -ForegroundColor Green
exit 0
