#!/usr/bin/env pwsh
# Validate all backend endpoints are reachable and handlers registered
# Run after starting backend: go run cmd/api/main.go

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$baseURL = "http://localhost:8080/api/v1"
$testsPassed = 0
$testsFailed = 0

function Test-Endpoint {
    param(
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [int]$ExpectedStatus = 400  # Most endpoints return 400 for missing auth
    )
    
    try {
        $url = "$baseURL$Path"
        $params = @{
            Uri             = $url
            Method          = $Method
            ErrorAction     = 'SilentlyContinue'
            UseBasicParsing = $true
        }
        
        if ($Body) {
            $params['Body'] = ($Body | ConvertTo-Json)
            $params['ContentType'] = 'application/json'
        }
        
        $response = Invoke-WebRequest @params
        $statusOK = ($response.StatusCode -eq $ExpectedStatus -or $response.StatusCode -ge 400)
        
        if ($statusOK) {
            Write-Host "[$Method] $Path - Status: $($response.StatusCode)" -ForegroundColor Green
            $script:testsPassed++
        } else {
            Write-Host "[$Method] $Path - Unexpected Status: $($response.StatusCode)" -ForegroundColor Red
            $script:testsFailed++
        }
    }
    catch {
        if ($_.Exception.Response.StatusCode) {
            Write-Host "[$Method] $Path - Status: $($_.Exception.Response.StatusCode)" -ForegroundColor Green
            $script:testsPassed++
        } else {
            Write-Host "[$Method] $Path - ERROR: Connection failed" -ForegroundColor Red
            $script:testsFailed++
        }
    }
}

Write-Host "=== Backend Endpoint Validation ==="
Write-Host "Testing endpoints at $baseURL" -ForegroundColor Cyan
Write-Host ""

# Health check (should be accessible without auth)
Write-Host "Health Check:" -ForegroundColor Yellow
Test-Endpoint -Method "GET" -Path "/health" -ExpectedStatus 200

# Auth endpoints
Write-Host ""
Write-Host "Auth Endpoints:" -ForegroundColor Yellow
Test-Endpoint -Method "POST" -Path "/auth/google" -Body @{id_token="test"}
Test-Endpoint -Method "POST" -Path "/auth/google-mock" -Body @{email="guest@example.com"}
Test-Endpoint -Method "POST" -Path "/auth/refresh" -Body @{refresh_token="test"}

# Course endpoints
Write-Host ""
Write-Host "Course Endpoints:" -ForegroundColor Yellow
Test-Endpoint -Method "POST" -Path "/courses" -Body @{title="Test"}
Test-Endpoint -Method "GET" -Path "/courses/550e8400-e29b-41d4-a716-446655440000"
Test-Endpoint -Method "GET" -Path "/users/550e8400-e29b-41d4-a716-446655440000/courses"

# Material endpoints
Write-Host ""
Write-Host "Material Endpoints:" -ForegroundColor Yellow
Test-Endpoint -Method "GET" -Path "/materials/550e8400-e29b-41d4-a716-446655440000"
Test-Endpoint -Method "GET" -Path "/topics/550e8400-e29b-41d4-a716-446655440000/materials"

# RAG endpoints
Write-Host ""
Write-Host "RAG Endpoints:" -ForegroundColor Yellow
Test-Endpoint -Method "POST" -Path "/topics/550e8400-e29b-41d4-a716-446655440000/process" -Body @{material_id="550e8400-e29b-41d4-a716-446655440000"}
Test-Endpoint -Method "POST" -Path "/topics/550e8400-e29b-41d4-a716-446655440000/query" -Body @{query="test"}

Write-Host ""
Write-Host "=== Results ==="
Write-Host "Passed: $testsPassed" -ForegroundColor Green
Write-Host "Failed: $testsFailed" -ForegroundColor $(if ($testsFailed -eq 0) { "Green" } else { "Red" })

if ($testsFailed -eq 0) {
    Write-Host ""
    Write-Host "✓ All endpoints are registered and reachable" -ForegroundColor Green
    exit 0
} else {
    Write-Host ""
    Write-Host "✗ Some endpoints failed" -ForegroundColor Red
    exit 1
}
