#!/bin/bash

# CityConnect API Integration Test Script
# This script tests all major API endpoints

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "=========================================="
echo "  CityConnect API Integration Tests"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo ""

# Test function
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local token="$5"
    local expected_status="$6"

    headers=""
    if [ -n "$token" ]; then
        headers="-H \"Authorization: Bearer $token\""
    fi

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $token" 2>/dev/null)
    fi

    body=$(echo "$response" | head -n -1)
    status=$(echo "$response" | tail -n 1)

    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} $name (HTTP $status)"
        echo "$body"
    else
        echo -e "${RED}✗${NC} $name - Expected $expected_status, got $status"
        echo "$body"
    fi
    echo ""
}

# Health Checks
echo "1. Health Checks"
echo "----------------"
test_endpoint "Gateway Health" "GET" "/health" "" "" "200"

# Auth Tests
echo "2. Authentication Tests"
echo "-----------------------"

# Register a new test user
TIMESTAMP=$(date +%s)
TEST_EMAIL="testuser_${TIMESTAMP}@test.com"

echo "Registering user: $TEST_EMAIL"
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"password123\",\"name\":\"Test User\",\"role\":\"warga\"}")
echo "Register response: $REGISTER_RESPONSE"
echo ""

# Login
echo "Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"password123\"}")
echo "Login response: $LOGIN_RESPONSE"

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo -e "${RED}Failed to get token. Using demo user...${NC}"
    LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"warga@test.com","password":"password123"}')
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi
echo "Token: ${TOKEN:0:20}..."
echo ""

# Report Tests
echo "3. Report Tests"
echo "---------------"

# Get public reports (no auth)
test_endpoint "Get Public Reports" "GET" "/api/v1/reports/public" "" "" "200"

# Get categories
test_endpoint "Get Categories" "GET" "/api/v1/reports/categories" "" "" "200"

# Create a report
echo "Creating a report..."
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/reports/" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
        "title": "Test Report - Integration Test",
        "description": "This is a test report created by integration test script",
        "category_id": 1,
        "privacy_level": "public"
    }')
echo "Create response: $CREATE_RESPONSE"
REPORT_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Report ID: $REPORT_ID"
echo ""

# Get my reports
test_endpoint "Get My Reports" "GET" "/api/v1/reports/my" "" "$TOKEN" "200"

# Vote Tests
echo "4. Vote Tests"
echo "-------------"

if [ -n "$REPORT_ID" ]; then
    # Upvote
    test_endpoint "Upvote Report" "POST" "/api/v1/reports/$REPORT_ID/vote" '{"vote_type":"upvote"}' "$TOKEN" "200"
    
    # Get vote status
    test_endpoint "Get Vote Status" "GET" "/api/v1/reports/$REPORT_ID/vote" "" "$TOKEN" "200"
    
    # Change to downvote
    test_endpoint "Change to Downvote" "POST" "/api/v1/reports/$REPORT_ID/vote" '{"vote_type":"downvote"}' "$TOKEN" "200"
    
    # Remove vote
    test_endpoint "Remove Vote" "DELETE" "/api/v1/reports/$REPORT_ID/vote" "" "$TOKEN" "200"
else
    echo -e "${RED}Skipping vote tests - no report ID${NC}"
fi
echo ""

# Notification Tests
echo "5. Notification Tests"
echo "---------------------"
test_endpoint "Get Notifications" "GET" "/api/v1/notifications/" "" "$TOKEN" "200"
test_endpoint "Mark All Read" "PATCH" "/api/v1/notifications/read-all" "" "$TOKEN" "200"

# Admin Tests
echo "6. Admin Tests"
echo "--------------"
echo "Logging in as admin..."
ADMIN_LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin_kebersihan@test.com","password":"password123"}')
ADMIN_TOKEN=$(echo "$ADMIN_LOGIN" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "Admin token: ${ADMIN_TOKEN:0:20}..."
echo ""

# Get reports as admin
test_endpoint "Admin Get Reports" "GET" "/api/v1/reports/" "" "$ADMIN_TOKEN" "200"

# Update status (if we have a report)
if [ -n "$REPORT_ID" ]; then
    test_endpoint "Update Report Status" "PATCH" "/api/v1/reports/$REPORT_ID/status" '{"status":"accepted"}' "$ADMIN_TOKEN" "200"
fi

echo "=========================================="
echo "  Integration Tests Complete"
echo "=========================================="
