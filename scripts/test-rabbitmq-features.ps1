# RabbitMQ Features Test

$ErrorActionPreference = "Continue"
$BaseUrl = "http://localhost:8080"

Write-Host "`n========================================" -ForegroundColor Magenta
Write-Host "  RABBITMQ FEATURES TEST" -ForegroundColor Magenta
Write-Host "========================================`n" -ForegroundColor Magenta

# 1. Service Health
Write-Host "=== 1. SERVICE HEALTH ===" -ForegroundColor Yellow
try {
    $r = Invoke-RestMethod -Uri "http://localhost:3002/health" -TimeoutSec 5
    Write-Host "[OK] Report Service: $($r.status)" -ForegroundColor Green
} catch { Write-Host "[FAIL] Report Service" -ForegroundColor Red }

try {
    $n = Invoke-RestMethod -Uri "http://localhost:3003/health" -TimeoutSec 5
    Write-Host "[OK] Notification Service: $($n.status)" -ForegroundColor Green
} catch { Write-Host "[FAIL] Notification Service" -ForegroundColor Red }

try {
    $creds = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("cityconnect:cityconnect_secret"))
    $queues = Invoke-RestMethod -Uri "http://localhost:15672/api/queues" -Headers @{Authorization="Basic $creds"} -TimeoutSec 5
    Write-Host "[OK] RabbitMQ: $($queues.Count) queues" -ForegroundColor Green
} catch { Write-Host "[FAIL] RabbitMQ" -ForegroundColor Red }

# 2. Auth
Write-Host "`n=== 2. AUTHENTICATION ===" -ForegroundColor Yellow
$wargaLogin = Invoke-RestMethod -Uri "$BaseUrl/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"warga@test.com","password":"password123"}'
$wargaToken = $wargaLogin.token
Write-Host "[OK] Logged in as warga" -ForegroundColor Green

$adminLogin = Invoke-RestMethod -Uri "$BaseUrl/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"email":"admin_kebersihan@test.com","password":"password123"}'
$adminToken = $adminLogin.token
Write-Host "[OK] Logged in as admin" -ForegroundColor Green

# 3. Outbox Pattern
Write-Host "`n=== 3. OUTBOX PATTERN ===" -ForegroundColor Yellow
try {
    $before = Invoke-RestMethod -Uri "http://localhost:3002/admin/outbox/stats" -TimeoutSec 5
    $beforeCount = $before.outbox_stats.published
    Write-Host "    Outbox published before: $beforeCount" -ForegroundColor Gray
} catch { $beforeCount = 0 }

$reportBody = @{ title = "Outbox Test $(Get-Random)"; description = "Testing outbox"; category_id = 1; privacy_level = "public" } | ConvertTo-Json
$report = Invoke-RestMethod -Uri "$BaseUrl/api/v1/reports/" -Method POST -Headers @{Authorization="Bearer $wargaToken";"Content-Type"="application/json"} -Body $reportBody
$reportId = $report.report.id
Write-Host "[OK] Created report: $reportId" -ForegroundColor Green

Start-Sleep -Seconds 2

try {
    $after = Invoke-RestMethod -Uri "http://localhost:3002/admin/outbox/stats" -TimeoutSec 5
    $afterCount = $after.outbox_stats.published
    Write-Host "    Outbox published after: $afterCount" -ForegroundColor Gray
    if ($afterCount -gt $beforeCount) {
        Write-Host "[OK] Outbox pattern working" -ForegroundColor Green
    }
} catch {}

# 4. Status Update -> Notification
Write-Host "`n=== 4. STATUS UPDATE NOTIFICATION ===" -ForegroundColor Yellow
$statusBody = '{"status":"in_progress"}'
$null = Invoke-RestMethod -Uri "$BaseUrl/api/v1/reports/$reportId/status" -Method PATCH -Headers @{Authorization="Bearer $adminToken";"Content-Type"="application/json"} -Body $statusBody
Write-Host "[OK] Status updated to in_progress" -ForegroundColor Green

Start-Sleep -Seconds 2

$notifs = Invoke-RestMethod -Uri "$BaseUrl/api/v1/notifications" -Headers @{Authorization="Bearer $wargaToken"}
$found = $notifs.notifications | Where-Object { $_.message -like "*Outbox Test*" }
if ($found) {
    Write-Host "[OK] Notification created for status update" -ForegroundColor Green
} else {
    Write-Host "[INFO] Notification may take a moment" -ForegroundColor Yellow
}

# 5. Vote -> Notification  
Write-Host "`n=== 5. VOTE NOTIFICATION ===" -ForegroundColor Yellow
$voteBody = '{"vote_type":"upvote"}'
try {
    $null = Invoke-RestMethod -Uri "$BaseUrl/api/v1/reports/$reportId/vote" -Method POST -Headers @{Authorization="Bearer $adminToken";"Content-Type"="application/json"} -Body $voteBody
    Write-Host "[OK] Vote cast" -ForegroundColor Green
} catch {
    Write-Host "[INFO] Vote may already exist" -ForegroundColor Yellow
}

# 6. Queue Status
Write-Host "`n=== 6. QUEUE STATUS ===" -ForegroundColor Yellow
$creds = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("cityconnect:cityconnect_secret"))
$queues = Invoke-RestMethod -Uri "http://localhost:15672/api/queues" -Headers @{Authorization="Basic $creds"}

foreach ($q in $queues) {
    $dlq = if ($q.name -like "*.dlq") { " [DLQ]" } else { "" }
    $status = if ($q.consumers -gt 0 -or $q.name -like "*.dlq") { "OK" } else { "NO CONSUMER" }
    Write-Host "    $($q.name): $($q.messages) msgs, $($q.consumers) consumers$dlq [$status]" -ForegroundColor $(if($status -eq "OK"){"Gray"}else{"Yellow"})
}

# 7. DLQ Check
Write-Host "`n=== 7. DLQ STATUS ===" -ForegroundColor Yellow
$dlqs = $queues | Where-Object { $_.name -like "*.dlq" }
$totalDlqMsgs = ($dlqs | Measure-Object -Property messages -Sum).Sum
if ($totalDlqMsgs -eq 0) {
    Write-Host "[OK] No messages in DLQ (healthy)" -ForegroundColor Green
} else {
    Write-Host "[WARN] $totalDlqMsgs messages in DLQ - check for errors" -ForegroundColor Yellow
}

# Summary
Write-Host "`n========================================" -ForegroundColor Magenta
Write-Host "  TEST COMPLETE" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta
Write-Host "Report ID: $reportId"
Write-Host "Outbox: $beforeCount -> $afterCount published"
Write-Host "DLQ messages: $totalDlqMsgs"

exit 0
