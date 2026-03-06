$env:GOOGLE_APPLICATION_CREDENTIALS = 'c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend\agent-klyra-75f3becb9bb4.json'
Write-Host '🔄 Iniciando Proxy...'
cd 'c:\Users\jeaqh\Desktop\Proyectos\Klyra\gemini-live-agent-klyra\backend\cmd\api'
& .\cloud-sql-proxy.exe agent-klyra:us-central1:klyra-db-pg | Write-Output
