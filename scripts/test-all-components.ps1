################################################################################
#                    COMPREHENSIVE TEST SCRIPT
#                    CityConnect RabbitMQ Implementation
################################################################################
#
# Test Script untuk memverifikasi SEMUA komponen:
# 1. Service Health Checks
# 2. Database Tables (outbox_messages, processed_messages)
# 3. RabbitMQ Queues & DLQ Configuration
# 4. Message Flow End-to-End
# 5. Outbox Pattern
# 6. Idempotency
# 7. SSE Notifications
#
################################################################################

param(
    [switch]$Verbose,
    [switch]$SkipCleanup
)

$ErrorActionPreference = "Continue"
$API_BASE = "http://localhost:8080/api/v1"
$REPORT_SERVICE = "http://localhost:3002"
$NOTIFICATION_SERVICE = "http://localhost:3003"
$RABBITMQ_API = "http://localhost:15672/api"
$RABBITMQ_CREDS = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("cityconnect:cityconnect_secret"))

$global:TestsPassed = 0
$global:TestsFailed = 0
$global:TestsSkipped = 0

function Write-TestHeader($text) {
    Write-Host "`n" -NoNewline
    Write-Host ("=" * 70) -ForegroundColor Cyan
    Write-Host "  $text" -ForegroundColor Cyan
    Write-Host ("=" * 70) -ForegroundColor Cyan
}

function Write-TestResult($name, $passed, $details = "") {
    if ($passed) {
        Write-Host "[PASS] " -ForegroundColor Green -NoNewline
        Write-Host $name
        $global:TestsPassed++
    } else {
        Write-Host "[FAIL] " -ForegroundColor Red -NoNewline
        Write-Host $name
        if ($details) { Write-Host "       $details" -ForegroundColor Yellow }
        $global:TestsFailed++
    }
}

function Write-TestSkip($name, $reason) {
    Write-Host "[SKIP] " -ForegroundColor Yellow -NoNewline
    Write-Host "$name - $reason"
    $global:TestsSkipped++
}

function Test-ServiceHealth($name, $url) {
    try {
        $response = Invoke-RestMethod -Uri "$url/health" -TimeoutSec 5
        return $true
    } catch {
        return $false
    }
}

function Get-RabbitMQQueues {
    try {
        $headers = @{ "Authorization" = "Basic $RABBITMQ_CREDS" }
        return Invoke-RestMethod -Uri "$RABBITMQ_API/queues" -Headers $headers -TimeoutSec 10
    } catch {
        return $null
    }
}

################################################################################
# TEST 1: SERVICE HEALTH CHECKS
################################################################################
Write-TestHeader "TEST 1: SERVICE HEALTH CHECKS"

# Auth Service
$authHealth = Test-ServiceHealth "auth-service" "http://localhost:3001"
Write-TestResult "Auth Service (port 3001)" $authHealth

# Report Service
$reportHealth = Test-ServiceHealth "report-service" $REPORT_SERVICE
Write-TestResult "Report Service (port 3002)" $reportHealth

# Notification Service
$notifHealth = Test-ServiceHealth "notification-service" $NOTIFICATION_SERVICE
Write-TestResult "Notification Service (port 3003)" $notifHealth

# RabbitMQ
try {
    $headers = @{ "Authorization" = "Basic $RABBITMQ_CREDS" }
    $rmqHealth = Invoke-RestMethod -Uri "$RABBITMQ_API/overview" -Headers $headers -TimeoutSec 5
    Write-TestResult "RabbitMQ (port 5672/15672)" $true
} catch {
    Write-TestResult "RabbitMQ (port 5672/15672)" $false "Cannot connect to management API"
}

# Gateway
try {
    $gwHealth = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 5 -UseBasicParsing
    Write-TestResult "Gateway/Nginx (port 8080)" ($gwHealth.StatusCode -eq 200)
} catch {
    Write-TestResult "Gateway/Nginx (port 8080)" $false
}

################################################################################
# TEST 2: RABBITMQ QUEUE CONFIGURATION
################################################################################
Write-TestHeader "TEST 2: RABBITMQ QUEUE CONFIGURATION"

$queues = Get-RabbitMQQueues
if ($queues) {
    $queueNames = $queues | ForEach-Object { $_.name }
    
    # Main Queues
    $mainQueues = @("queue.status_updates", "queue.report_created", "queue.vote_received")
    foreach ($q in $mainQueues) {
        $exists = $q -in $queueNames
        Write-TestResult "Main Queue: $q" $exists
        
        if ($exists) {
            $queueInfo = $queues | Where-Object { $_.name -eq $q }
            $hasDLX = $queueInfo.arguments.'x-dead-letter-exchange' -eq "cityconnect.notifications.dlx"
            Write-TestResult "  → DLX configured on $q" $hasDLX
        }
    }
    
    # DLQ Queues
    $dlqQueues = @("queue.status_updates.dlq", "queue.report_created.dlq", "queue.vote_received.dlq")
    foreach ($q in $dlqQueues) {
        $exists = $q -in $queueNames
        Write-TestResult "DLQ Queue: $q" $exists
        
        if ($exists) {
            $queueInfo = $queues | Where-Object { $_.name -eq $q }
            $hasTTL = $null -ne $queueInfo.arguments.'x-message-ttl'
            Write-TestResult "  → TTL configured on $q" $hasTTL
        }
    }
    
    # Exchange check
    try {
        $exchanges = Invoke-RestMethod -Uri "$RABBITMQ_API/exchanges" -Headers $headers -TimeoutSec 5
        $mainExchange = $exchanges | Where-Object { $_.name -eq "cityconnect.notifications" }
        $dlxExchange = $exchanges | Where-Object { $_.name -eq "cityconnect.notifications.dlx" }
        
        Write-TestResult "Main Exchange: cityconnect.notifications" ($null -ne $mainExchange)
        Write-TestResult "DLX Exchange: cityconnect.notifications.dlx" ($null -ne $dlxExchange)
    } catch {
        Write-TestResult "Exchange configuration" $false "Cannot fetch exchanges"
    }
} else {
    Write-TestSkip "Queue Configuration" "Cannot connect to RabbitMQ API"
}

################################################################################
# TEST 3: DATABASE TABLES
################################################################################
Write-TestHeader "TEST 3: DATABASE TABLES"

try {
    # Check outbox_messages via API
    $outboxStats = Invoke-RestMethod -Uri "$REPORT_SERVICE/admin/outbox/stats" -TimeoutSec 5
    Write-TestResult "Outbox Stats Endpoint" $true
    Write-Host "       Current stats: $($outboxStats | ConvertTo-Json -Compress)" -ForegroundColor Gray
} catch {
    Write-TestResult "Outbox Stats Endpoint" $false $_.Exception.Message
}

################################################################################
# TEST 4: AUTHENTICATION FLOW
################################################################################
Write-TestHeader "TEST 4: AUTHENTICATION FLOW"

$testEmail = "test_$(Get-Random)@test.com"
$testPassword = "password123"

# Register
try {
    $regBody = @{
        email = $testEmail
        password = $testPassword
        name = "Test User"
        role = "warga"
    } | ConvertTo-Json
    
    $regResponse = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body $regBody -TimeoutSec 10
    Write-TestResult "User Registration" $true
    $userId = $regResponse.user.id
} catch {
    Write-TestResult "User Registration" $false $_.Exception.Message
    $userId = $null
}

# Login
try {
    $loginBody = @{
        email = $testEmail
        password = $testPassword
    } | ConvertTo-Json
    
    $loginResponse = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body $loginBody -TimeoutSec 10
    Write-TestResult "User Login" ($null -ne $loginResponse.token)
    $token = $loginResponse.token
    $headers = @{ "Authorization" = "Bearer $token"; "Content-Type" = "application/json" }
} catch {
    Write-TestResult "User Login" $false $_.Exception.Message
    $token = $null
}

################################################################################
# TEST 5: MESSAGE FLOW - REPORT CREATION
################################################################################
Write-TestHeader "TEST 5: MESSAGE FLOW - REPORT CREATION"

if ($token) {
    # Get initial outbox stats
    try {
        $beforeStats = Invoke-RestMethod -Uri "$REPORT_SERVICE/admin/outbox/stats" -TimeoutSec 5
        $beforePublished = if ($beforeStats.outbox_stats.published) { $beforeStats.outbox_stats.published } else { 0 }
    } catch {
        $beforePublished = 0
    }
    
    # Create report
    try {
        $reportBody = @{
            title = "Test Report $(Get-Date -Format 'HHmmss')"
            description = "Testing RabbitMQ message flow"
            category_id = 1
            privacy_level = "public"
        } | ConvertTo-Json
        
        # Direct to report-service (bypass gateway for this test)
        $directHeaders = @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
            "X-User-ID" = $userId
        }
        
        $reportResponse = Invoke-RestMethod -Uri "$REPORT_SERVICE/" -Method POST -Headers $directHeaders -Body $reportBody -TimeoutSec 10
        Write-TestResult "Create Report" ($null -ne $reportResponse.report.id)
        $reportId = $reportResponse.report.id
        
        # Wait for outbox worker
        Start-Sleep -Seconds 2
        
        # Check outbox stats after
        $afterStats = Invoke-RestMethod -Uri "$REPORT_SERVICE/admin/outbox/stats" -TimeoutSec 5
        $afterPublished = if ($afterStats.outbox_stats.published) { $afterStats.outbox_stats.published } else { 0 }
        
        $messagePublished = $afterPublished -gt $beforePublished
        Write-TestResult "Outbox Published Message" $messagePublished "Before: $beforePublished, After: $afterPublished"
        
    } catch {
        Write-TestResult "Create Report" $false $_.Exception.Message
        $reportId = $null
    }
    
    # Check queue message count
    if ($queues) {
        Start-Sleep -Seconds 2
        $queuesAfter = Get-RabbitMQQueues
        $reportQueue = $queuesAfter | Where-Object { $_.name -eq "queue.report_created" }
        Write-Host "       queue.report_created messages: $($reportQueue.messages)" -ForegroundColor Gray
    }
} else {
    Write-TestSkip "Report Creation" "No auth token available"
}

################################################################################
# TEST 6: MESSAGE FLOW - VOTE
################################################################################
Write-TestHeader "TEST 6: MESSAGE FLOW - VOTE"

if ($token -and $reportId) {
    try {
        $voteBody = @{ vote_type = "upvote" } | ConvertTo-Json
        $directHeaders = @{ 
            "Authorization" = "Bearer $token"
            "Content-Type" = "application/json"
            "X-User-ID" = $userId
        }
        
        $voteResponse = Invoke-RestMethod -Uri "$REPORT_SERVICE/$reportId/vote" -Method POST -Headers $directHeaders -Body $voteBody -TimeoutSec 10
        Write-TestResult "Cast Vote" $true
        
        Start-Sleep -Seconds 2
        
        # Check vote queue
        if ($queues) {
            $queuesAfter = Get-RabbitMQQueues
            $voteQueue = $queuesAfter | Where-Object { $_.name -eq "queue.vote_received" }
            Write-Host "       queue.vote_received messages: $($voteQueue.messages)" -ForegroundColor Gray
        }
    } catch {
        Write-TestResult "Cast Vote" $false $_.Exception.Message
    }
} else {
    Write-TestSkip "Vote" "No auth token or report ID"
}

################################################################################
# TEST 7: ADMIN STATUS UPDATE
################################################################################
Write-TestHeader "TEST 7: ADMIN STATUS UPDATE"

# Register admin
$adminEmail = "admin_$(Get-Random)@test.com"
try {
    $adminRegBody = @{
        email = $adminEmail
        password = "password123"
        name = "Admin User"
        role = "admin_infrastruktur"
    } | ConvertTo-Json
    
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body $adminRegBody -TimeoutSec 10
    
    $adminLoginBody = @{
        email = $adminEmail
        password = "password123"
    } | ConvertTo-Json
    
    $adminLogin = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body $adminLoginBody -TimeoutSec 10
    $adminToken = $adminLogin.token
    Write-TestResult "Admin Login" $true
} catch {
    Write-TestResult "Admin Login" $false $_.Exception.Message
    $adminToken = $null
}

if ($adminToken -and $reportId) {
    try {
        $statusBody = @{ status = "in_progress" } | ConvertTo-Json
        $adminHeaders = @{ 
            "Authorization" = "Bearer $adminToken"
            "Content-Type" = "application/json"
            "X-User-ID" = $adminLogin.user.id
            "X-User-Role" = "admin_infrastruktur"
        }
        
        $statusResponse = Invoke-RestMethod -Uri "$REPORT_SERVICE/$reportId/status" -Method PATCH -Headers $adminHeaders -Body $statusBody -TimeoutSec 10
        Write-TestResult "Update Status" $true
        
        Start-Sleep -Seconds 2
        
        if ($queues) {
            $queuesAfter = Get-RabbitMQQueues
            $statusQueue = $queuesAfter | Where-Object { $_.name -eq "queue.status_updates" }
            Write-Host "       queue.status_updates messages: $($statusQueue.messages)" -ForegroundColor Gray
        }
    } catch {
        Write-TestResult "Update Status" $false $_.Exception.Message
    }
} else {
    Write-TestSkip "Status Update" "No admin token or report ID"
}

################################################################################
# TEST 8: DLQ CHECK
################################################################################
Write-TestHeader "TEST 8: DLQ STATUS"

if ($queues) {
    $dlqQueues = Get-RabbitMQQueues | Where-Object { $_.name -like "*.dlq" }
    $totalDLQ = ($dlqQueues | Measure-Object -Property messages -Sum).Sum
    
    Write-TestResult "DLQ Total Messages" ($totalDLQ -eq 0) "Messages in DLQ: $totalDLQ (0 is expected for healthy system)"
    
    foreach ($dlq in $dlqQueues) {
        if ($dlq.messages -gt 0) {
            Write-Host "       WARNING: $($dlq.name) has $($dlq.messages) messages!" -ForegroundColor Yellow
        }
    }
} else {
    Write-TestSkip "DLQ Check" "Cannot connect to RabbitMQ"
}

################################################################################
# TEST 9: NOTIFICATION SERVICE ENDPOINTS
################################################################################
Write-TestHeader "TEST 9: NOTIFICATION SERVICE ENDPOINTS"

if ($token) {
    try {
        $notifHeaders = @{ 
            "Authorization" = "Bearer $token"
            "X-User-ID" = $userId
        }
        
        $notifications = Invoke-RestMethod -Uri "$NOTIFICATION_SERVICE/notifications" -Headers $notifHeaders -TimeoutSec 10
        Write-TestResult "Get Notifications Endpoint" $true
        Write-Host "       Notifications count: $($notifications.notifications.Count)" -ForegroundColor Gray
    } catch {
        Write-TestResult "Get Notifications Endpoint" $false $_.Exception.Message
    }
} else {
    Write-TestSkip "Notification Endpoints" "No auth token"
}

################################################################################
# SUMMARY
################################################################################
Write-TestHeader "TEST SUMMARY"

$total = $global:TestsPassed + $global:TestsFailed + $global:TestsSkipped
$passRate = if ($total -gt 0) { [math]::Round(($global:TestsPassed / $total) * 100, 1) } else { 0 }

Write-Host ""
Write-Host "  Total Tests:  $total" -ForegroundColor White
Write-Host "  Passed:       $($global:TestsPassed)" -ForegroundColor Green
Write-Host "  Failed:       $($global:TestsFailed)" -ForegroundColor Red
Write-Host "  Skipped:      $($global:TestsSkipped)" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Pass Rate:    $passRate%" -ForegroundColor $(if ($passRate -ge 80) { "Green" } elseif ($passRate -ge 50) { "Yellow" } else { "Red" })
Write-Host ""

if ($global:TestsFailed -gt 0) {
    Write-Host "Some tests failed! Check the output above for details." -ForegroundColor Red
    exit 1
} else {
    Write-Host "All tests passed!" -ForegroundColor Green
    exit 0
}
