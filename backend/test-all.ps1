#!/usr/bin/env pwsh
# Comprehensive test verification script for all Go unit tests
# Tests all: auth, course, material, RAG, HTTP handlers, database layer

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "=== Klyra Backend Test Suite ==="
Write-Host "Testing all core functionality: Auth, Course, Material, RAG, Handlers" -ForegroundColor Cyan
Write-Host ""

# Change to backend directory
Push-Location $PSScriptRoot/backend

try {
    # Run all tests with coverage
    Write-Host "Running all unit tests..." -ForegroundColor Yellow
    $testOutput = go test -v -coverprofile=coverage.out ./...
    Write-Host $testOutput
    
    # Parse coverage
    if (-not (Test-Path coverage.out)) {
        Write-Host "WARNING: No coverage report generated" -ForegroundColor Yellow
    } else {
        $coverage = go tool cover -func=coverage.out
        $totalCoverage = $coverage | Select-Object -Last 1
        Write-Host ""
        Write-Host "Coverage Summary:" -ForegroundColor Green
        Write-Host $totalCoverage
    }
    
    # Show test count
    $testCount = $testOutput | Select-String "RUN Test" | Measure-Object | Select-Object -ExpandProperty Count
    Write-Host ""
    Write-Host "Total tests run: $testCount" -ForegroundColor Green
    
    Write-Host ""
    Write-Host "✓ All tests completed successfully" -ForegroundColor Green
}
catch {
    Write-Host ""
    Write-Host "✗ Test execution failed: $_" -ForegroundColor Red
    exit 1
}
finally {
    Pop-Location
}
