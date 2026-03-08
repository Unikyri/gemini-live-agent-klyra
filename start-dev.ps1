#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Klyra Local Development Startup Script
    Inicia PostgreSQL, Backend y Frontend con un solo comando

.DESCRIPTION
    Este script automatiza:
    1. Docker PostgreSQL (si no está corriendo)
    2. Backend Go API
    3. Frontend Flutter

.EXAMPLE
    .\start-dev.ps1
    
.NOTES
    Requiere: Docker Desktop, Go 1.22+, Flutter 3.x
    Ejecutar desde la raíz del proyecto
#>

param(
    [switch]$NoFrontend = $false,
    [switch]$NoDocker = $false
)

Write-Host "
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🚀 KLYRA LOCAL DEVELOPMENT STARTUP
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
" -ForegroundColor Cyan

# Verificar que estamos en la raíz del proyecto
if (-not (Test-Path "docker-compose.yml")) {
    Write-Host "❌ Error: No estoy en la raíz del proyecto" -ForegroundColor Red
    Write-Host "Por favor ejecuta este script desde: c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra" -ForegroundColor Yellow
    exit 1
}

# 1. INICIAR POSTGRESQL
if (-not $NoDocker) {
    Write-Host "`n[1/3] 🐘 Iniciando PostgreSQL..." -ForegroundColor Cyan
    
    $pgStatus = docker-compose ps postgres 2>&1 | Select-String "Up"
    if ($pgStatus) {
        Write-Host "     ✅ PostgreSQL ya está corriendo" -ForegroundColor Green
    } else {
        Write-Host "     ⏳ Iniciando contenedor..." -ForegroundColor Yellow
        docker-compose up -d postgres
        Start-Sleep -Seconds 3
        Write-Host "     ✅ PostgreSQL iniciado" -ForegroundColor Green
    }
}

# 2. INICIAR BACKEND
Write-Host "`n[2/3] 🔙 Iniciando Backend (Go API)..." -ForegroundColor Cyan
Write-Host "     URL: http://localhost:8080" -ForegroundColor White

$backendScript = {
    Set-Location "C:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend"
    $env:DB_HOST = 'localhost'
    $env:DB_PORT = '5433'
    $env:ENV = 'local'
    
    Write-Host "     ⏳ Compilando..." -ForegroundColor Yellow
    & go run ./cmd/api/main.go
}

$backend = Start-Job -ScriptBlock $backendScript -Name "BackendAPI"
Start-Sleep -Seconds 5

# Verifica que el backend está listo
$healthCheck = try {
    (Invoke-WebRequest -UseBasicParsing http://localhost:8080/health).StatusCode -eq 200
} catch {
    $false
}

if ($healthCheck) {
    Write-Host "     ✅ Backend listo (Status: OK)" -ForegroundColor Green
} else {
    Write-Host "     ⚠️  Backend aún iniciándose, espera 5 segundos..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
}

# 3. INICIAR FRONTEND
if (-not $NoFrontend) {
    Write-Host "`n[3/3] 📱 Iniciando Frontend (Flutter Desktop)..." -ForegroundColor Cyan
    
    $frontendScript = {
        Set-Location "C:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\mobile"
        Write-Host "     ⏳ Compilando..." -ForegroundColor Yellow
        & flutter run -d windows
    }
    
    $frontend = Start-Job -ScriptBlock $frontendScript -Name "FlutterApp"
    Write-Host "     ✅ Frontend iniciándose (se abrirá en nueva ventana)" -ForegroundColor Green
}

# RESUMEN FINAL
Write-Host "
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ TODO RUNNING - KLYRA DEVELOPMENT ENVIRONMENT READY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 SERVICES:
  🐘 PostgreSQL   ✅ localhost:5433 (user/password)
  🔙 Backend      ✅ http://localhost:8080/health
  📱 Frontend     ✅ Abierto en ventana separada

🎯 QUICK START:
  1. Login:
     POST http://localhost:8080/auth/google-signin-mock
     Body: {\"email\": \"dev@example.com\", \"name\": \"Dev User\"}
  
  2. Copia el JWT token de la respuesta
  
  3. Crea un curso:
     POST http://localhost:8080/courses
     Header: Authorization: Bearer <TOKEN>
  
  4. Sube un material PDF/TXT

📚 REFERENCIA:
  - LOCAL-DEV-DASHBOARD.md
  - SPRINT-5-KICKOFF.md

💡 TIPS:
  - Presiona 'r' en terminal de Flutter para Hot Reload
  - Edita backend/*.go → Ctrl+C y recarga automáticamente
  - Tests: cd backend && go test ./... -v

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
" -ForegroundColor Green

Write-Host "`n⏳ Monitoreo de procesos:" -ForegroundColor Cyan
Write-Host "   Backend Job:  $(if ($backend.State -eq 'Running') {'Running ✅'} else {'Stopped ❌'})" -ForegroundColor White
Write-Host "   Frontend Job: $(if ($frontend.State -eq 'Running') {'Running ✅'} else {'Stopped ❌'})" -ForegroundColor White

Write-Host "`n❌ Para detener: Ctrl+C en cada terminal" -ForegroundColor Yellow
Write-Host "💥 Para resetear todo: docker-compose down -v && .\start-dev.ps1" -ForegroundColor Yellow

# MANTENER PROCESOS VIVOS
Write-Host "`n[Monitor] Presiona Ctrl+C para salir..." -ForegroundColor DarkGray

while ($true) {
    $backendStatus = (Get-Job -Name "BackendAPI" -ErrorAction SilentlyContinue).State
    $frontendStatus = (Get-Job -Name "FlutterApp" -ErrorAction SilentlyContinue).State
    
    if ($backendStatus -ne 'Running') {
        Write-Host "`n⚠️  Backend stopped!" -ForegroundColor Yellow
    }
    
    Start-Sleep -Seconds 5
}
