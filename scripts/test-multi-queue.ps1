# Load Test: Multiple Queues RabbitMQ

$API_BASE = "http://localhost:8080/api/v1"

# Login/Register
Write-Host "`n[SETUP] Preparing test user..." -ForegroundColor Cyan
try {
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body '{"email":"mqtest@test.com","password":"password123","name":"MQ Tester","role":"warga"}' -ErrorAction SilentlyContinue
} catch {}

$login = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"mqtest@test.com","password":"password123"}'
$TOKEN = $login.token
$headers = @{ "Authorization" = "Bearer $TOKEN"; "Content-Type" = "application/json" }

Write-Host "[OK] Token obtained" -ForegroundColor Green

# Admin login
try {
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body '{"email":"adminmq@test.com","password":"password123","name":"Admin MQ","role":"admin_infrastruktur"}' -ErrorAction SilentlyContinue
} catch {}
$adminLogin = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"adminmq@test.com","password":"password123"}'
$ADMIN_TOKEN = $adminLogin.token
$adminHeaders = @{ "Authorization" = "Bearer $ADMIN_TOKEN"; "Content-Type" = "application/json" }

Write-Host "[OK] Admin token obtained" -ForegroundColor Green

# PHASE 1: Create Reports
Write-Host "`nPHASE 1: CREATE REPORTS" -ForegroundColor Yellow

$reportIds = @()
$count = 30

Write-Host "Creating $count reports..."
$sw = [System.Diagnostics.Stopwatch]::StartNew()

for ($i = 1; $i -le $count; $i++) {
    $body = @{
        title = "MQ Test Report $i"
        description = "Load test report $i"
        category_id = 7
        privacy_level = "public"
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$API_BASE/reports/" -Method POST -Headers $headers -Body $body
        $reportIds += $response.report.id
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()
Write-Host "`n[OK] Created $($reportIds.Count) reports in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green

Write-Host "`n[WAIT] 8 seconds..." -ForegroundColor Magenta
Start-Sleep -Seconds 8

# PHASE 2: Cast Votes
Write-Host "`nPHASE 2: CAST VOTES" -ForegroundColor Yellow

Write-Host "Voting on $($reportIds.Count) reports..."
$sw.Restart()

foreach ($id in $reportIds) {
    $voteType = if ((Get-Random -Maximum 2) -eq 0) { "upvote" } else { "downvote" }
    $body = @{ vote_type = $voteType } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Uri "$API_BASE/reports/$id/vote" -Method POST -Headers $headers -Body $body
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()
Write-Host "`n[OK] Voted on $($reportIds.Count) reports in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green

Write-Host "`n[WAIT] 8 seconds..." -ForegroundColor Magenta
Start-Sleep -Seconds 8

# PHASE 3: Update Status
Write-Host "`nPHASE 3: UPDATE STATUS" -ForegroundColor Yellow

$statuses = @("accepted", "in_progress", "completed")

Write-Host "Updating status on $($reportIds.Count) reports..."
$sw.Restart()

foreach ($id in $reportIds) {
    $status = $statuses | Get-Random
    $body = @{ status = $status } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Uri "$API_BASE/reports/$id/status" -Method PATCH -Headers $adminHeaders -Body $body
        Write-Host "." -NoNewline
    } catch {
        Write-Host "x" -NoNewline
    }
}
$sw.Stop()
Write-Host "`n[OK] Updated $($reportIds.Count) statuses in $($sw.ElapsedMilliseconds)ms" -ForegroundColor Green

# Summary
Write-Host "`nLOAD TEST COMPLETE" -ForegroundColor Cyan
Write-Host "Total: $($count * 3) ops (reports: $count, votes: $count, status: $count)"
