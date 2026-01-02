# Extreme Load Test: Multiple Queues (100 ops per phase)

$API_BASE = "http://localhost:8080/api/v1"

# Login/Register
Write-Host "`n[SETUP] Preparing test users..." -ForegroundColor Cyan
try {
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body '{"email":"extreme@test.com","password":"password123","name":"Extreme Tester","role":"warga"}' -ErrorAction SilentlyContinue
} catch {}

$login = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"extreme@test.com","password":"password123"}'
$TOKEN = $login.token
$headers = @{ "Authorization" = "Bearer $TOKEN"; "Content-Type" = "application/json" }

Write-Host "[OK] User token obtained" -ForegroundColor Green

# Admin login
try {
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body '{"email":"adminextreme@test.com","password":"password123","name":"Admin Extreme","role":"admin_infrastruktur"}' -ErrorAction SilentlyContinue
} catch {}
$adminLogin = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"adminextreme@test.com","password":"password123"}'
$ADMIN_TOKEN = $adminLogin.token
$adminHeaders = @{ "Authorization" = "Bearer $ADMIN_TOKEN"; "Content-Type" = "application/json" }

Write-Host "[OK] Admin token obtained" -ForegroundColor Green

$count = 100

# PHASE 1: Create Reports
Write-Host "`nPHASE 1: CREATE $count REPORTS" -ForegroundColor Red

$reportIds = [System.Collections.ArrayList]::new()
$sw = [System.Diagnostics.Stopwatch]::StartNew()

$jobs = @()
for ($i = 1; $i -le $count; $i++) {
    $body = @{
        title = "Extreme Test $i - $(Get-Random)"
        description = "Load test report $i"
        category_id = 7
        privacy_level = "public"
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$API_BASE/reports/" -Method POST -Headers $headers -Body $body -TimeoutSec 30
        [void]$reportIds.Add($response.report.id)
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()

Write-Host ""
Write-Host "[OK] Created $($reportIds.Count)/$count reports in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green
Write-Host "    Rate: $([math]::Round($reportIds.Count / ($sw.ElapsedMilliseconds / 1000), 2)) req/sec" -ForegroundColor Yellow

Write-Host "`n[WAIT] 10 seconds..." -ForegroundColor Magenta
Start-Sleep -Seconds 10

# PHASE 2: Votes
Write-Host "`nPHASE 2: VOTE $($reportIds.Count) REPORTS" -ForegroundColor Red

$sw.Restart()
$voteCount = 0

foreach ($id in $reportIds) {
    $voteType = if ((Get-Random -Maximum 2) -eq 0) { "upvote" } else { "downvote" }
    $body = @{ vote_type = $voteType } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Uri "$API_BASE/reports/$id/vote" -Method POST -Headers $headers -Body $body -TimeoutSec 30
        $voteCount++
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()

Write-Host ""
Write-Host "[OK] Voted on $voteCount/$($reportIds.Count) reports in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green
Write-Host "    Rate: $([math]::Round($voteCount / ($sw.ElapsedMilliseconds / 1000), 2)) req/sec" -ForegroundColor Yellow

Write-Host "`n[WAIT] 10 seconds..." -ForegroundColor Magenta
Start-Sleep -Seconds 10

# PHASE 3: Status Updates
Write-Host "`nPHASE 3: UPDATE $($reportIds.Count) STATUSES" -ForegroundColor Red

$statuses = @("accepted", "in_progress", "completed")
$sw.Restart()
$statusCount = 0

foreach ($id in $reportIds) {
    $status = $statuses | Get-Random
    $body = @{ status = $status } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Uri "$API_BASE/reports/$id/status" -Method PATCH -Headers $adminHeaders -Body $body -TimeoutSec 30
        $statusCount++
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()

Write-Host ""
Write-Host "[OK] Updated $statusCount/$($reportIds.Count) statuses in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green
Write-Host "    Rate: $([math]::Round($statusCount / ($sw.ElapsedMilliseconds / 1000), 2)) req/sec" -ForegroundColor Yellow

# Queue stats
try {
    $queues = Invoke-RestMethod -Uri "http://localhost:15672/api/queues" -Headers @{Authorization = "Basic " + [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("cityconnect:cityconnect_secret"))}
    Write-Host "`nQUEUE STATUS:" -ForegroundColor Cyan
    foreach ($q in $queues) {
        Write-Host "  $($q.name): $($q.messages) msgs, $($q.consumers) consumers"
    }
} catch {}

# Summary
$total = $reportIds.Count + $voteCount + $statusCount
Write-Host "`nEXTREME LOAD TEST COMPLETE - $total total ops" -ForegroundColor Cyan
