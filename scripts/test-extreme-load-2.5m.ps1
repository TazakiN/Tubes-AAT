################################################################################
#                    EXTREME LOAD TEST - 2.5M USERS TARGET
#                    CityConnect RabbitMQ Stress Test
################################################################################

param(
    [int]$Users = 1000,
    [int]$ReportsPerUser = 10,
    [int]$VotesPerReport = 5,
    [int]$BatchSize = 50,
    [int]$WarmupSeconds = 10,
    [switch]$SkipWarmup,
    [switch]$Insane
)

$ErrorActionPreference = "Continue"

# MODE GILA untuk target 2.5M users
if ($Insane) {
    $Users = 10000
    $ReportsPerUser = 25
    $VotesPerReport = 10
    $BatchSize = 200
    Write-Host "INSANE MODE ACTIVATED!" -ForegroundColor Red
}

$API_BASE = "http://localhost:8080/api/v1"
$REPORT_SERVICE = "http://localhost:3002"
$RABBITMQ_API = "http://localhost:15672/api"
$RABBITMQ_CREDS = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("cityconnect:cityconnect_secret"))

# Statistics
$global:Stats = @{
    UsersCreated = 0
    UsersFailed = 0
    ReportsCreated = 0
    ReportsFailed = 0
    VotesCast = 0
    VotesFailed = 0
    StatusUpdates = 0
    StatusFailed = 0
    TotalRequests = 0
    TotalErrors = 0
    StartTime = $null
    EndTime = $null
}

function Write-Banner {
    param($text, $color = "Cyan")
    Write-Host ""
    Write-Host ("=" * 80) -ForegroundColor $color
    Write-Host "  $text" -ForegroundColor $color
    Write-Host ("=" * 80) -ForegroundColor $color
}

function Write-Progress2 {
    param($current, $total, $text)
    $percent = [math]::Round(($current / $total) * 100, 1)
    $bar = "#" * [math]::Floor($percent / 2) + "-" * (50 - [math]::Floor($percent / 2))
    Write-Host "`r[$bar] $percent% - $text" -NoNewline
}

function Get-RabbitMQStats {
    try {
        $headers = @{ "Authorization" = "Basic $RABBITMQ_CREDS" }
        $overview = Invoke-RestMethod -Uri "$RABBITMQ_API/overview" -Headers $headers -TimeoutSec 5
        $queues = Invoke-RestMethod -Uri "$RABBITMQ_API/queues" -Headers $headers -TimeoutSec 5
        
        return @{
            MessagesTotal = ($queues | Measure-Object -Property messages -Sum).Sum
            MessagesReady = ($queues | Measure-Object -Property messages_ready -Sum).Sum
            MessagesUnacked = ($queues | Measure-Object -Property messages_unacknowledged -Sum).Sum
            PublishRate = $overview.message_stats.publish_details.rate
            DeliverRate = $overview.message_stats.deliver_get_details.rate
            Connections = $overview.object_totals.connections
        }
    } catch {
        return $null
    }
}

function Register-TestUser {
    param($index)
    
    $email = "loadtest_${index}_$(Get-Random)@test.com"
    $body = @{
        email = $email
        password = "password123"
        name = "Load Test User $index"
        role = "warga"
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body $body -TimeoutSec 30
        $loginBody = @{ email = $email; password = "password123" } | ConvertTo-Json
        $loginResponse = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body $loginBody -TimeoutSec 30
        
        return @{
            Success = $true
            UserId = $response.user.id
            Token = $loginResponse.token
            Email = $email
        }
    } catch {
        return @{ Success = $false }
    }
}

function Create-TestReport {
    param($token, $userId, $index)
    
    $headers = @{ 
        "Authorization" = "Bearer $token"
        "Content-Type" = "application/json"
        "X-User-ID" = $userId
    }
    
    $body = @{
        title = "Load Test Report $index - $(Get-Random)"
        description = "High load stress test report for 2.5M user simulation"
        category_id = (1..8 | Get-Random)
        privacy_level = "public"  # Must be public to allow voting
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$REPORT_SERVICE/" -Method POST -Headers $headers -Body $body -TimeoutSec 30
        return @{ Success = $true; ReportId = $response.report.id }
    } catch {
        return @{ Success = $false }
    }
}

function Cast-TestVote {
    param($token, $userId, $reportId)
    
    $headers = @{ 
        "Authorization" = "Bearer $token"
        "Content-Type" = "application/json"
        "X-User-ID" = $userId
    }
    
    $body = @{ vote_type = @("upvote", "downvote") | Get-Random } | ConvertTo-Json
    
    try {
        $null = Invoke-RestMethod -Uri "$REPORT_SERVICE/$reportId/vote" -Method POST -Headers $headers -Body $body -TimeoutSec 30 -ErrorAction Stop
        return $true
    } catch {
        Write-Host "Vote error for report $reportId : $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

################################################################################
# MAIN EXECUTION
################################################################################

Clear-Host
Write-Host ""
Write-Host "================================================================================" -ForegroundColor Magenta
Write-Host "               EXTREME LOAD TEST - TARGET 2.5 MILLION USERS                    " -ForegroundColor Magenta
Write-Host "================================================================================" -ForegroundColor Magenta
Write-Host ""

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  Concurrent Users:    $Users"
Write-Host "  Reports per User:    $ReportsPerUser"
Write-Host "  Votes per Report:    $VotesPerReport"
Write-Host "  Batch Size:          $BatchSize"
Write-Host ""
Write-Host "Expected Operations:" -ForegroundColor Yellow
$totalReports = $Users * $ReportsPerUser
$maxVotersPerReport = [math]::Min($VotesPerReport, $Users - 1)  # Can't vote on own report
$totalVotes = $totalReports * $maxVotersPerReport
$totalOps = $Users + $totalReports + $totalVotes
Write-Host "  Total Users:         $Users"
Write-Host "  Total Reports:       $totalReports"
Write-Host "  Total Votes:         $totalVotes (max $maxVotersPerReport per report, $($Users-1) voters available)"
Write-Host "  Total Operations:    $totalOps"
Write-Host ""

# Check RabbitMQ stats before
Write-Banner "PRE-TEST RABBITMQ STATUS"
$beforeStats = Get-RabbitMQStats
if ($beforeStats) {
    Write-Host "Messages in Queues:  $($beforeStats.MessagesTotal)"
    Write-Host "Connections:         $($beforeStats.Connections)"
} else {
    Write-Host "Cannot connect to RabbitMQ!" -ForegroundColor Red
}

# Warmup
if (-not $SkipWarmup) {
    Write-Banner "WARMUP PHASE ($WarmupSeconds seconds)"
    for ($i = $WarmupSeconds; $i -gt 0; $i--) {
        Write-Host "`rStarting in $i seconds..." -NoNewline
        Start-Sleep -Seconds 1
    }
    Write-Host ""
}

$global:Stats.StartTime = Get-Date
$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

################################################################################
# PHASE 1: USER REGISTRATION (PS 5.1 Compatible - Sequential)
################################################################################
Write-Banner "PHASE 1: REGISTERING $Users USERS" "Green"

$userList = New-Object System.Collections.ArrayList

for ($idx = 1; $idx -le $Users; $idx++) {
    if ($idx % 10 -eq 0 -or $idx -eq $Users) {
        Write-Progress2 $idx $Users "Registering user $idx of $Users"
    }
    
    $email = "loadtest_${idx}_$(Get-Random)@test.com"
    $body = @{
        email = $email
        password = "password123"
        name = "Load Test User $idx"
        role = "warga"
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body $body -TimeoutSec 30
        $loginBody = @{ email = $email; password = "password123" } | ConvertTo-Json
        $loginResponse = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body $loginBody -TimeoutSec 30
        
        $null = $userList.Add(@{ Success = $true; UserId = $response.user.id; Token = $loginResponse.token })
        $global:Stats.UsersCreated++
    } catch {
        $global:Stats.UsersFailed++
    }
    $global:Stats.TotalRequests += 2
}

Write-Host ""
Write-Host "Users registered: $($global:Stats.UsersCreated)/$Users" -ForegroundColor $(if ($global:Stats.UsersCreated -eq $Users) { "Green" } else { "Yellow" })

################################################################################
# PHASE 2: REPORT CREATION
################################################################################
Write-Banner "PHASE 2: CREATING $totalReports REPORTS" "Blue"

$reportList = New-Object System.Collections.ArrayList
$userArray = $userList.ToArray()

$reportCount = 0
foreach ($user in $userArray) {
    for ($r = 0; $r -lt $ReportsPerUser; $r++) {
        $reportCount++
        
        if ($reportCount % $BatchSize -eq 0 -or $reportCount -eq $totalReports) {
            Write-Progress2 $reportCount $totalReports "Creating reports..."
        }
        
        $result = Create-TestReport -token $user.Token -userId $user.UserId -index $reportCount
        
        if ($result.Success) {
            $null = $reportList.Add(@{ ReportId = $result.ReportId; ReporterToken = $user.Token; ReporterUserId = $user.UserId })
            $global:Stats.ReportsCreated++
        } else {
            $global:Stats.ReportsFailed++
        }
        $global:Stats.TotalRequests++
    }
}

Write-Host ""
Write-Host "Reports created: $($global:Stats.ReportsCreated)/$totalReports" -ForegroundColor $(if ($global:Stats.ReportsCreated -eq $totalReports) { "Green" } else { "Yellow" })

# Check RabbitMQ
$midStats = Get-RabbitMQStats
if ($midStats) {
    Write-Host "RabbitMQ - Messages in queue: $($midStats.MessagesTotal), Publish rate: $($midStats.PublishRate)/s" -ForegroundColor Cyan
}

################################################################################
# PHASE 3: VOTING STORM
################################################################################
Write-Banner "PHASE 3: VOTING STORM" "Yellow"

$reportArray = $reportList.ToArray()
$voteAttempts = 0
$actualMaxVotes = 0

foreach ($report in $reportArray) {
    # Get voters excluding the reporter
    $availableVoters = @($userArray | Where-Object { $_.UserId -ne $report.ReporterUserId })
    $votersToUse = [math]::Min($VotesPerReport, $availableVoters.Count)
    $actualMaxVotes += $votersToUse
    
    if ($availableVoters.Count -eq 0) { continue }
    
    $voters = $availableVoters | Get-Random -Count $votersToUse
    
    foreach ($voter in $voters) {
        $voteAttempts++
        
        if ($voteAttempts % ($BatchSize * 2) -eq 0 -or $voteAttempts -eq $actualMaxVotes) {
            Write-Progress2 $voteAttempts $actualMaxVotes "Casting votes..."
        }
        
        $success = Cast-TestVote -token $voter.Token -userId $voter.UserId -reportId $report.ReportId
        
        if ($success) {
            $global:Stats.VotesCast++
        } else {
            $global:Stats.VotesFailed++
        }
        $global:Stats.TotalRequests++
    }
}

Write-Host ""
$voteStatus = if ($global:Stats.VotesFailed -eq 0) { "Green" } else { "Yellow" }
Write-Host "Votes cast: $($global:Stats.VotesCast)/$voteAttempts" -ForegroundColor $voteStatus
if ($voteAttempts -lt $totalVotes) {
    Write-Host "  (Limited by available voters: $($Users-1) users can vote per report)" -ForegroundColor DarkGray
}

################################################################################
# PHASE 4: STATUS UPDATE STORM (Admin operations)
################################################################################
Write-Banner "PHASE 4: STATUS UPDATES" "Red"

$adminEmail = "admin_load_$(Get-Random)@test.com"
$adminBody = @{
    email = $adminEmail
    password = "password123"
    name = "Load Test Admin"
    role = "admin_infrastruktur"
} | ConvertTo-Json

$adminToken = $null
$adminUserId = $null

try {
    $null = Invoke-RestMethod -Uri "$API_BASE/auth/register" -Method POST -ContentType "application/json" -Body $adminBody -TimeoutSec 30
    $adminLogin = Invoke-RestMethod -Uri "$API_BASE/auth/login" -Method POST -ContentType "application/json" -Body (@{ email = $adminEmail; password = "password123" } | ConvertTo-Json) -TimeoutSec 30
    $adminToken = $adminLogin.token
    $adminUserId = $adminLogin.user.id
    Write-Host "Admin user created" -ForegroundColor Green
} catch {
    Write-Host "Failed to create admin" -ForegroundColor Red
}

if ($adminToken) {
    $statuses = @("accepted", "in_progress", "completed", "rejected")
    $statusCount = 0
    $totalStatusUpdates = [math]::Min($reportArray.Count, 1000)
    
    foreach ($report in ($reportArray | Select-Object -First $totalStatusUpdates)) {
        $statusCount++
        
        if ($statusCount % 50 -eq 0 -or $statusCount -eq $totalStatusUpdates) {
            Write-Progress2 $statusCount $totalStatusUpdates "Updating statuses..."
        }
        
        $headers = @{ 
            "Authorization" = "Bearer $adminToken"
            "Content-Type" = "application/json"
            "X-User-ID" = $adminUserId
            "X-User-Role" = "admin_infrastruktur"
        }
        
        $body = @{ status = ($statuses | Get-Random) } | ConvertTo-Json
        
        try {
            $null = Invoke-RestMethod -Uri "$REPORT_SERVICE/$($report.ReportId)/status" -Method PATCH -Headers $headers -Body $body -TimeoutSec 30
            $global:Stats.StatusUpdates++
        } catch {
            $global:Stats.StatusFailed++
        }
        $global:Stats.TotalRequests++
    }
    
    Write-Host ""
    Write-Host "Status updates: $($global:Stats.StatusUpdates)/$totalStatusUpdates" -ForegroundColor $(if ($global:Stats.StatusFailed -eq 0) { "Green" } else { "Yellow" })
}

################################################################################
# RESULTS
################################################################################
$stopwatch.Stop()
$global:Stats.EndTime = Get-Date

Write-Host ""
Write-Host ""
Write-Banner "LOAD TEST COMPLETE" "Magenta"

$duration = $stopwatch.Elapsed
$totalOps = $global:Stats.TotalRequests
$opsPerSec = [math]::Round($totalOps / $duration.TotalSeconds, 2)

Write-Host ""
Write-Host "================================================================================" -ForegroundColor White
Write-Host "                              RESULTS SUMMARY                                   " -ForegroundColor White
Write-Host "================================================================================" -ForegroundColor White
Write-Host ""
Write-Host "  Duration:              $($duration.ToString('hh\:mm\:ss\.fff'))"
Write-Host "  Total Requests:        $totalOps"
Write-Host "  Throughput:            $opsPerSec req/sec"
Write-Host ""
Write-Host "  Users Created:         $($global:Stats.UsersCreated) (Failed: $($global:Stats.UsersFailed))"
Write-Host "  Reports Created:       $($global:Stats.ReportsCreated) (Failed: $($global:Stats.ReportsFailed))"
Write-Host "  Votes Cast:            $($global:Stats.VotesCast) (Failed: $($global:Stats.VotesFailed))"
Write-Host "  Status Updates:        $($global:Stats.StatusUpdates) (Failed: $($global:Stats.StatusFailed))"
Write-Host ""
Write-Host "================================================================================" -ForegroundColor White

# RabbitMQ Final Stats
Write-Banner "POST-TEST RABBITMQ STATUS" "Cyan"
$afterStats = Get-RabbitMQStats
if ($afterStats) {
    Write-Host "Messages in Queues:    $($afterStats.MessagesTotal)"
    Write-Host "Messages Ready:        $($afterStats.MessagesReady)"
    Write-Host "Messages Unacked:      $($afterStats.MessagesUnacked)"
    Write-Host "Connections:           $($afterStats.Connections)"
    
    if ($afterStats.PublishRate) {
        Write-Host "Publish Rate:          $($afterStats.PublishRate)/sec"
    }
    if ($afterStats.DeliverRate) {
        Write-Host "Deliver Rate:          $($afterStats.DeliverRate)/sec"
    }
}

# Check DLQ
try {
    $headers = @{ "Authorization" = "Basic $RABBITMQ_CREDS" }
    $queues = Invoke-RestMethod -Uri "$RABBITMQ_API/queues" -Headers $headers -TimeoutSec 5
    $dlqMessages = ($queues | Where-Object { $_.name -like "*.dlq" } | Measure-Object -Property messages -Sum).Sum
    
    Write-Host ""
    if ($dlqMessages -gt 0) {
        Write-Host "WARNING: $dlqMessages messages in DLQ!" -ForegroundColor Yellow
    } else {
        Write-Host "DLQ is empty (no failed messages)" -ForegroundColor Green
    }
} catch {
    Write-Host "Could not check DLQ status" -ForegroundColor Yellow
}

# Outbox stats
try {
    $outboxStats = Invoke-RestMethod -Uri "$REPORT_SERVICE/admin/outbox/stats" -TimeoutSec 5
    Write-Host ""
    Write-Host "Outbox Stats: $($outboxStats.outbox_stats | ConvertTo-Json -Compress)" -ForegroundColor Cyan
} catch {
    Write-Host "Could not check outbox stats" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "================================================================================" -ForegroundColor Magenta

# Scaling estimation
$usersPerSec = [math]::Round($global:Stats.UsersCreated / $duration.TotalSeconds, 2)
$timeFor2_5M = [math]::Round(2500000 / $opsPerSec / 3600, 2)
$recommendedReplicas = [math]::Ceiling($timeFor2_5M / 0.5)

Write-Host ""
Write-Host "SCALING ESTIMATION FOR 2.5M USERS:" -ForegroundColor Yellow
Write-Host "  Current throughput:     $opsPerSec ops/sec"
Write-Host "  Time for 2.5M users:    ~$timeFor2_5M hours (single instance)"
Write-Host "  Recommended scaling:    $recommendedReplicas notification-service replicas for under 30min processing"
Write-Host ""

if ($Insane) {
    Write-Host "INSANE MODE TEST COMPLETE!" -ForegroundColor Green
}

Write-Host "Test complete! Check the results above." -ForegroundColor Cyan
