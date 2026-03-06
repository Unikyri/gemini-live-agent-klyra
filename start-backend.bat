@echo off
REM Script para lanzar Klyra Backend con Cloud SQL Proxy
REM ====================================================

setlocal enabledelayedexpansion

REM Configurar credenciales GCP
set "GOOGLE_APPLICATION_CREDENTIALS=c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend\agent-klyra-75f3becb9bb4.json"

echo.
echo ========================================
echo  INICIANDO KLYRA BACKEND
echo ========================================
echo.

REM Matar procesos antiguos
echo [1/4] Limpiando procesos anteriores...
taskkill /F /IM cloud-sql-proxy.exe 2>nul
taskkill /F /IM go.exe 2>nul
timeout /t 1 /nobreak >nul

REM Iniciar Cloud SQL Proxy
echo [2/4] Iniciando Cloud SQL Proxy...
cd /d "c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend\cmd\api"
start "Cloud SQL Proxy" cloud-sql-proxy.exe agent-klyra:us-central1:klyra-db-pg
timeout /t 4 /nobreak >nul

REM Iniciar Backend
echo [3/4] Iniciando Backend Go...
cd /d "c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend"
start "Klyra Backend" go run ./cmd/api/main.go
timeout /t 3 /nobreak >nul

REM Verificar estado
echo [4/4] Verificando conexion...
powershell -NoProfile -Command "try { $r = Invoke-WebRequest http://localhost:8080/health -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop; Write-Host 'A! ^[32mBACKEND ONLINE^[0m - Status: '$r.StatusCode } catch { Write-Host 'Started... check http://localhost:8080/health in 5 seconds' }"

echo.
echo ========================================
echo  KLYRA BACKEND LANZADO
echo ========================================
echo.
echo URL: http://localhost:8080
echo Health Check: http://localhost:8080/health
echo.
pause
