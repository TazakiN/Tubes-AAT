# CityConnect API Test Script (PowerShell)
# Usage: .\test-api.ps1

$BASE_URL = "http://localhost:8080/api/v1"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "CityConnect API Test" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# Test 1: Health Check
Write-Host "`n1. Gateway Health Check..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
    Write-Host "Gateway OK" -ForegroundColor Green
} catch {
    Write-Host "Gateway not ready: $_" -ForegroundColor Red
    exit 1
}

# Test 2: Login Warga (using seed data)
Write-Host "`n2. Login Warga (warga@test.com)..." -ForegroundColor Yellow
$loginBody = @{
    email = "warga@test.com"
    password = "password123"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "$BASE_URL/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
    $wargaToken = $loginResponse.token
    Write-Host "Login successful!" -ForegroundColor Green
    Write-Host "User: $($loginResponse.user.name)"
} catch {
    Write-Host "Login failed: $_" -ForegroundColor Red
}

# Test 3: Create Public Report
Write-Host "`n3. Create Public Report (as Warga)..." -ForegroundColor Yellow
$reportBody = @{
    title = "Sampah menumpuk di Jalan ABC"
    description = "Sudah 5 hari tidak diangkut, bau menyengat"
    category_id = 2
    location_lat = -6.2088
    location_lng = 106.8456
    privacy_level = "public"
} | ConvertTo-Json

try {
    $headers = @{ Authorization = "Bearer $wargaToken" }
    $reportResponse = Invoke-RestMethod -Uri "$BASE_URL/reports/" -Method Post -Body $reportBody -ContentType "application/json" -Headers $headers
    Write-Host "Report created!" -ForegroundColor Green
    Write-Host "Report ID: $($reportResponse.report.id)"
} catch {
    Write-Host "Create report failed: $_" -ForegroundColor Red
}

# Test 4: Create Anonymous Report
Write-Host "`n4. Create Anonymous Report..." -ForegroundColor Yellow
$anonReportBody = @{
    title = "Laporan korupsi"
    description = "Detail rahasia tentang korupsi"
    category_id = 1
    privacy_level = "anonymous"
} | ConvertTo-Json

try {
    $anonReportResponse = Invoke-RestMethod -Uri "$BASE_URL/reports/" -Method Post -Body $anonReportBody -ContentType "application/json" -Headers $headers
    $anonReportId = $anonReportResponse.report.id
    Write-Host "Anonymous report created!" -ForegroundColor Green
    Write-Host "Report ID: $anonReportId"
} catch {
    Write-Host "Create anonymous report failed: $_" -ForegroundColor Red
}

# Test 5: Login Admin Kebersihan
Write-Host "`n5. Login Admin Kebersihan..." -ForegroundColor Yellow
$adminLoginBody = @{
    email = "admin_kebersihan@test.com"
    password = "password123"
} | ConvertTo-Json

try {
    $adminLoginResponse = Invoke-RestMethod -Uri "$BASE_URL/auth/login" -Method Post -Body $adminLoginBody -ContentType "application/json"
    $adminToken = $adminLoginResponse.token
    Write-Host "Admin login successful!" -ForegroundColor Green
} catch {
    Write-Host "Admin login failed: $_" -ForegroundColor Red
}

# Test 6: Get Reports (Admin Kebersihan - should only see kebersihan)
Write-Host "`n6. Get Reports (as Admin Kebersihan)..." -ForegroundColor Yellow
try {
    $adminHeaders = @{ Authorization = "Bearer $adminToken" }
    $reportsResponse = Invoke-RestMethod -Uri "$BASE_URL/reports/" -Method Get -Headers $adminHeaders
    Write-Host "Total reports: $($reportsResponse.total)" -ForegroundColor Green
    
    foreach ($report in $reportsResponse.reports) {
        Write-Host "  - [$($report.status)] $($report.title) (Category: $($report.category.department))"
    }
} catch {
    Write-Host "Get reports failed: $_" -ForegroundColor Red
}

# Test 7: View Anonymous Report Detail (should hide reporter)
Write-Host "`n7. View Anonymous Report (reporter should be hidden)..." -ForegroundColor Yellow
if ($anonReportId) {
    try {
        $anonDetailResponse = Invoke-RestMethod -Uri "$BASE_URL/reports/$anonReportId" -Method Get -Headers $adminHeaders
        
        if ($null -eq $anonDetailResponse.reporter_id) {
            Write-Host "[PASS] reporter_id is hidden (null)" -ForegroundColor Green
        } else {
            Write-Host "[FAIL] reporter_id should be null for anonymous reports" -ForegroundColor Red
        }
    } catch {
        Write-Host "Get anonymous report failed: $_" -ForegroundColor Red
    }
}

# Test 8: Login Admin Kesehatan and verify RBAC
Write-Host "`n8. Test RBAC: Admin Kesehatan should NOT see Kebersihan reports..." -ForegroundColor Yellow
$kesehatanLoginBody = @{
    email = "admin_kesehatan@test.com"
    password = "password123"
} | ConvertTo-Json

try {
    $kesehatanLoginResponse = Invoke-RestMethod -Uri "$BASE_URL/auth/login" -Method Post -Body $kesehatanLoginBody -ContentType "application/json"
    $kesehatanToken = $kesehatanLoginResponse.token
    
    $kesehatanHeaders = @{ Authorization = "Bearer $kesehatanToken" }
    $kesehatanReportsResponse = Invoke-RestMethod -Uri "$BASE_URL/reports/" -Method Get -Headers $kesehatanHeaders
    
    $hasKebersihan = $false
    foreach ($report in $kesehatanReportsResponse.reports) {
        if ($report.category.department -eq "kebersihan") {
            $hasKebersihan = $true
            break
        }
    }
    
    if ($hasKebersihan) {
        Write-Host "[FAIL] Admin Kesehatan can see Kebersihan reports!" -ForegroundColor Red
    } else {
        Write-Host "[PASS] RBAC working - Admin Kesehatan cannot see Kebersihan reports" -ForegroundColor Green
    }
} catch {
    Write-Host "RBAC test failed: $_" -ForegroundColor Red
}

Write-Host "`n==========================================" -ForegroundColor Cyan
Write-Host "Test Complete!" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
