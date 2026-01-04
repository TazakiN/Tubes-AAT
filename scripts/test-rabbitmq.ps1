# =============================================================================
# RabbitMQ STRESS TEST - CityConnect
# =============================================================================

param(
    [int]$ReportCount = 20,
    [int]$DelayMs = 30
)

$baseUrl = "http://localhost:8080/api/v1"

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "          RABBITMQ STRESS TEST - CITYCONNECT                " -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  Reports: $ReportCount" -ForegroundColor White
Write-Host "  Expected RabbitMQ Messages:" -ForegroundColor White
Write-Host "    - report.created:        $ReportCount messages" -ForegroundColor Yellow
Write-Host "    - report.vote.received:  $($ReportCount * 2) messages" -ForegroundColor Yellow
Write-Host "    - report.status.updated: $($ReportCount * 3) messages" -ForegroundColor Yellow
Write-Host "  TOTAL: $($ReportCount + ($ReportCount * 2) + ($ReportCount * 3)) messages" -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

# =============================================================================
# STEP 1: LOGIN
# =============================================================================
Write-Host "[STEP 1/5] Logging in users..." -ForegroundColor Magenta

$wargaBody = @{email="warga@test.com"; password="password123"} | ConvertTo-Json
$wargaToken = (Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -ContentType "application/json" -Body $wargaBody).token

$adminBody = @{email="admin_kebersihan@test.com"; password="password123"} | ConvertTo-Json
$adminToken = (Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -ContentType "application/json" -Body $adminBody).token

$admin2Body = @{email="admin_kesehatan@test.com"; password="password123"} | ConvertTo-Json
$admin2Token = (Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method POST -ContentType "application/json" -Body $admin2Body).token

Write-Host "  [OK] 3 users logged in" -ForegroundColor Green

# =============================================================================
# STEP 2: CREATE REPORTS (triggers report.created)
# =============================================================================
Write-Host ""
Write-Host "[STEP 2/5] Creating $ReportCount reports..." -ForegroundColor Magenta
Write-Host "  >>> WATCH: Message rates 'Publish' should spike <<<" -ForegroundColor Yellow

$reportIds = [System.Collections.ArrayList]@()
$categories = @(1, 2, 3, 7)

for ($i = 1; $i -le $ReportCount; $i++) {
    $catId = $categories[($i - 1) % $categories.Count]
    $body = @{
        title = "Stress Test Report #$i"
        description = "Load test report number $i"
        category_id = $catId
        privacy_level = "public"
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$baseUrl/reports/" -Method POST `
            -Headers @{Authorization="Bearer $wargaToken"} `
            -ContentType "application/json" -Body $body
        
        [void]$reportIds.Add($response.report.id)
        Write-Host "  Created report #$i" -ForegroundColor Gray
        
        Start-Sleep -Milliseconds $DelayMs
    } catch {
        Write-Host "  Error creating report #$i" -ForegroundColor Red
    }
}

Write-Host "  [OK] $($reportIds.Count) reports created = $($reportIds.Count) messages to RabbitMQ" -ForegroundColor Green

Write-Host ""
Write-Host "  Waiting 5 seconds before voting (so you can see separate spikes)..." -ForegroundColor Cyan
Start-Sleep -Seconds 5

# =============================================================================
# STEP 3: VOTE ALL REPORTS (triggers report.vote.received)
# =============================================================================
Write-Host ""
Write-Host "[STEP 3/5] Voting on all reports (2 voters each)..." -ForegroundColor Magenta
Write-Host "  >>> WATCH: Another spike in 'Publish' rate <<<" -ForegroundColor Yellow

$voteCount = 0
$voteBody = @{vote_type="upvote"} | ConvertTo-Json

foreach ($reportId in $reportIds) {
    # Vote dari admin1
    try {
        Invoke-RestMethod -Uri "$baseUrl/reports/$reportId/vote" -Method POST `
            -Headers @{Authorization="Bearer $adminToken"} `
            -ContentType "application/json" -Body $voteBody | Out-Null
        $voteCount++
    } catch {}
    
    # Vote dari admin2
    try {
        Invoke-RestMethod -Uri "$baseUrl/reports/$reportId/vote" -Method POST `
            -Headers @{Authorization="Bearer $admin2Token"} `
            -ContentType "application/json" -Body $voteBody | Out-Null
        $voteCount++
    } catch {}
    
    Write-Host "  Voted on report $($reportIds.IndexOf($reportId) + 1)/$($reportIds.Count)" -ForegroundColor Gray
    Start-Sleep -Milliseconds $DelayMs
}

Write-Host "  [OK] $voteCount votes cast = $voteCount messages to RabbitMQ" -ForegroundColor Green

Write-Host ""
Write-Host "  Waiting 5 seconds before status updates (so you can see separate spikes)..." -ForegroundColor Cyan
Start-Sleep -Seconds 5

# =============================================================================
# STEP 4: UPDATE STATUS (triggers report.status.updated)
# =============================================================================
Write-Host ""
Write-Host "[STEP 4/5] Updating status (3 transitions each)..." -ForegroundColor Magenta
Write-Host "  >>> WATCH: Biggest spike! 3x messages <<<" -ForegroundColor Yellow

$statuses = @("accepted", "in_progress", "completed")
$statusCount = 0

foreach ($reportId in $reportIds) {
    foreach ($status in $statuses) {
        try {
            $statusBody = @{status=$status} | ConvertTo-Json
            Invoke-RestMethod -Uri "$baseUrl/reports/$reportId/status" -Method PATCH `
                -Headers @{Authorization="Bearer $adminToken"} `
                -ContentType "application/json" -Body $statusBody | Out-Null
            $statusCount++
        } catch {}
        
        Start-Sleep -Milliseconds $DelayMs
    }
    Write-Host "  Updated report $($reportIds.IndexOf($reportId) + 1)/$($reportIds.Count) through 3 statuses" -ForegroundColor Gray
}

Write-Host "  [OK] $statusCount status updates = $statusCount messages to RabbitMQ" -ForegroundColor Green

# =============================================================================
# STEP 5: SUMMARY
# =============================================================================
$totalMessages = $reportIds.Count + $voteCount + $statusCount

Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "                    TEST COMPLETED!                         " -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
Write-Host "  Messages sent to RabbitMQ:" -ForegroundColor White
Write-Host "    - report.created:        $($reportIds.Count) messages" -ForegroundColor White
Write-Host "    - report.vote.received:  $voteCount messages" -ForegroundColor White
Write-Host "    - report.status.updated: $statusCount messages" -ForegroundColor White
Write-Host "    ----------------------------------------" -ForegroundColor Gray
Write-Host "    TOTAL:                   $totalMessages messages" -ForegroundColor Yellow
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "WHAT TO CHECK IN RABBITMQ UI (http://localhost:15672):" -ForegroundColor Cyan
Write-Host ""
Write-Host "  1. OVERVIEW TAB:" -ForegroundColor Yellow
Write-Host "     - Message rates chart: should show spikes during test"
Write-Host "     - Queued messages Ready: should be 0 (all processed)"
Write-Host "     - Publish rate was active during test"
Write-Host ""
Write-Host "  2. QUEUES TAB:" -ForegroundColor Yellow
Write-Host "     - notification.events queue"
Write-Host "     - Ready: 0, Unacked: 0 (all consumed)"
Write-Host "     - Click queue name to see detailed charts"
Write-Host ""
Write-Host "  3. CONNECTIONS TAB:" -ForegroundColor Yellow
Write-Host "     - 1 connection from report-service"
Write-Host ""
Write-Host "  4. CHANNELS TAB:" -ForegroundColor Yellow
Write-Host "     - 1 channel with consumer"
Write-Host ""
Write-Host "Run: docker logs report-service --tail 50" -ForegroundColor Gray
Write-Host ""
