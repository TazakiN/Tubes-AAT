#!/bin/bash

# CityConnect API Test Script
# Usage: ./test-api.sh

BASE_URL="http://localhost:8080/api/v1"

echo "=========================================="
echo "CityConnect API Test"
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Register Warga
echo ""
echo "1. Register Warga..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testwarga@test.com",
    "password": "password123",
    "name": "Test Warga",
    "role": "warga"
  }')
echo "Response: $REGISTER_RESPONSE"

# Test 2: Login Warga
echo ""
echo "2. Login Warga..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "warga@test.com",
    "password": "password123"
  }')
echo "Response: $LOGIN_RESPONSE"

# Extract token (requires jq)
WARGA_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | sed 's/"token":"//')
echo "Token: $WARGA_TOKEN"

# Test 3: Create Public Report
echo ""
echo "3. Create Public Report (as Warga)..."
CREATE_REPORT=$(curl -s -X POST "$BASE_URL/reports/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WARGA_TOKEN" \
  -d '{
    "title": "Sampah menumpuk di depan rumah",
    "description": "Sudah seminggu tidak diangkut",
    "category_id": 2,
    "location_lat": -6.2088,
    "location_lng": 106.8456,
    "privacy_level": "public"
  }')
echo "Response: $CREATE_REPORT"

# Test 4: Create Anonymous Report
echo ""
echo "4. Create Anonymous Report (as Warga)..."
ANON_REPORT=$(curl -s -X POST "$BASE_URL/reports/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WARGA_TOKEN" \
  -d '{
    "title": "Korupsi di dinas kebersihan",
    "description": "Detail rahasia",
    "category_id": 1,
    "privacy_level": "anonymous"
  }')
echo "Response: $ANON_REPORT"
ANON_REPORT_ID=$(echo $ANON_REPORT | grep -o '"id":"[^"]*' | sed 's/"id":"//')

# Test 5: Login Admin Kebersihan
echo ""
echo "5. Login Admin Kebersihan..."
ADMIN_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin_kebersihan@test.com",
    "password": "password123"
  }')
ADMIN_TOKEN=$(echo $ADMIN_LOGIN | grep -o '"token":"[^"]*' | sed 's/"token":"//')
echo "Token: $ADMIN_TOKEN"

# Test 6: Get Reports (as Admin Kebersihan - should only see kebersihan category)
echo ""
echo "6. Get Reports (as Admin Kebersihan)..."
ADMIN_REPORTS=$(curl -s "$BASE_URL/reports/" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
echo "Response: $ADMIN_REPORTS"

# Test 7: Get Anonymous Report Detail (should hide reporter info)
echo ""
echo "7. Get Anonymous Report Detail..."
if [ ! -z "$ANON_REPORT_ID" ]; then
  ANON_DETAIL=$(curl -s "$BASE_URL/reports/$ANON_REPORT_ID" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
  echo "Response: $ANON_DETAIL"
  
  # Check if reporter_id is null/hidden
  if echo "$ANON_DETAIL" | grep -q '"reporter_id":null'; then
    echo -e "${GREEN}✓ Anonymous report: reporter_id is hidden${NC}"
  else
    echo -e "${RED}✗ Anonymous report: reporter_id should be null${NC}"
  fi
fi

# Test 8: Login Admin Kesehatan and try to access Kebersihan reports
echo ""
echo "8. Test RBAC: Admin Kesehatan tries to see Kebersihan reports..."
KESEHATAN_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin_kesehatan@test.com",
    "password": "password123"
  }')
KESEHATAN_TOKEN=$(echo $KESEHATAN_LOGIN | grep -o '"token":"[^"]*' | sed 's/"token":"//')

KESEHATAN_REPORTS=$(curl -s "$BASE_URL/reports/" \
  -H "Authorization: Bearer $KESEHATAN_TOKEN")
echo "Response: $KESEHATAN_REPORTS"

# Should NOT contain kebersihan reports
if echo "$KESEHATAN_REPORTS" | grep -q '"department":"kebersihan"'; then
  echo -e "${RED}✗ RBAC Failed: Kesehatan admin can see Kebersihan reports${NC}"
else
  echo -e "${GREEN}✓ RBAC Success: Kesehatan admin cannot see Kebersihan reports${NC}"
fi

echo ""
echo "=========================================="
echo "Test Complete!"
echo "=========================================="
