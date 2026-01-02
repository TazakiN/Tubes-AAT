#!/bin/bash
# Extreme Load Test: Multiple Queues (100 ops per phase)

API_BASE="http://localhost:8080/api/v1"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "\n${CYAN}[SETUP] Preparing test users...${NC}"

# Register user (ignore errors if already exists)
curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"extreme@test.com","password":"password123","name":"Extreme Tester","role":"warga"}' > /dev/null 2>&1

# Login user
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"extreme@test.com","password":"password123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}[ERROR] Failed to get user token${NC}"
  exit 1
fi
echo -e "${GREEN}[OK] User token obtained${NC}"

# Register admin (ignore errors if already exists)
curl -s -X POST "$API_BASE/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"adminextreme@test.com","password":"password123","name":"Admin Extreme","role":"admin_infrastruktur"}' > /dev/null 2>&1

# Login admin
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"adminextreme@test.com","password":"password123"}')

ADMIN_TOKEN=$(echo "$ADMIN_LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo -e "${RED}[ERROR] Failed to get admin token${NC}"
  exit 1
fi
echo -e "${GREEN}[OK] Admin token obtained${NC}"

COUNT=100
REPORT_IDS=()

# PHASE 1: Create Reports
echo -e "\n${RED}PHASE 1: CREATE $COUNT REPORTS${NC}"

START_TIME=$(date +%s%3N)
CREATE_SUCCESS=0

for i in $(seq 1 $COUNT); do
  RANDOM_NUM=$RANDOM
  BODY=$(cat <<EOF
{"title":"Extreme Test $i - $RANDOM_NUM","description":"Load test report $i","category_id":7,"privacy_level":"public"}
EOF
)
  
  RESPONSE=$(curl -s -X POST "$API_BASE/reports/" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$BODY" \
    --max-time 30)
  
  REPORT_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  
  if [ -n "$REPORT_ID" ]; then
    REPORT_IDS+=("$REPORT_ID")
    CREATE_SUCCESS=$((CREATE_SUCCESS + 1))
    echo -n "."
  else
    echo -n "x"
  fi
done

END_TIME=$(date +%s%3N)
ELAPSED=$((END_TIME - START_TIME))
RATE=$(awk "BEGIN {printf \"%.2f\", $CREATE_SUCCESS / ($ELAPSED / 1000)}")

echo ""
echo -e "${GREEN}[OK] Created $CREATE_SUCCESS/$COUNT reports in ${ELAPSED}ms${NC}"
echo -e "${YELLOW}    Rate: $RATE req/sec${NC}"

echo -e "\n${MAGENTA}[WAIT] 10 seconds...${NC}"
sleep 10

# PHASE 2: Votes
REPORT_COUNT=${#REPORT_IDS[@]}
echo -e "\n${RED}PHASE 2: VOTE $REPORT_COUNT REPORTS${NC}"

START_TIME=$(date +%s%3N)
VOTE_SUCCESS=0

for id in "${REPORT_IDS[@]}"; do
  if [ $((RANDOM % 2)) -eq 0 ]; then
    VOTE_TYPE="upvote"
  else
    VOTE_TYPE="downvote"
  fi
  
  RESPONSE=$(curl -s -X POST "$API_BASE/reports/$id/vote" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"vote_type\":\"$VOTE_TYPE\"}" \
    --max-time 30)
  
  if echo "$RESPONSE" | grep -q "vote_score"; then
    VOTE_SUCCESS=$((VOTE_SUCCESS + 1))
    echo -n "."
  else
    echo -n "x"
  fi
done

END_TIME=$(date +%s%3N)
ELAPSED=$((END_TIME - START_TIME))
RATE=$(awk "BEGIN {printf \"%.2f\", $VOTE_SUCCESS / ($ELAPSED / 1000)}")

echo ""
echo -e "${GREEN}[OK] Voted on $VOTE_SUCCESS/$REPORT_COUNT reports in ${ELAPSED}ms${NC}"
echo -e "${YELLOW}    Rate: $RATE req/sec${NC}"

echo -e "\n${MAGENTA}[WAIT] 10 seconds...${NC}"
sleep 10

# PHASE 3: Status Updates
echo -e "\n${RED}PHASE 3: UPDATE $REPORT_COUNT STATUSES${NC}"

STATUSES=("accepted" "in_progress" "completed")
START_TIME=$(date +%s%3N)
STATUS_SUCCESS=0

for id in "${REPORT_IDS[@]}"; do
  STATUS=${STATUSES[$((RANDOM % 3))]}
  
  RESPONSE=$(curl -s -X PATCH "$API_BASE/reports/$id/status" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"status\":\"$STATUS\"}" \
    --max-time 30)
  
  if echo "$RESPONSE" | grep -q "message"; then
    STATUS_SUCCESS=$((STATUS_SUCCESS + 1))
    echo -n "."
  else
    echo -n "x"
  fi
done

END_TIME=$(date +%s%3N)
ELAPSED=$((END_TIME - START_TIME))
RATE=$(awk "BEGIN {printf \"%.2f\", $STATUS_SUCCESS / ($ELAPSED / 1000)}")

echo ""
echo -e "${GREEN}[OK] Updated $STATUS_SUCCESS/$REPORT_COUNT statuses in ${ELAPSED}ms${NC}"
echo -e "${YELLOW}    Rate: $RATE req/sec${NC}"

# Queue stats
echo -e "\n${CYAN}QUEUE STATUS:${NC}"
QUEUE_RESPONSE=$(curl -s -u "cityconnect:cityconnect_secret" "http://localhost:15672/api/queues" 2>/dev/null)

if [ -n "$QUEUE_RESPONSE" ]; then
  echo "$QUEUE_RESPONSE" | grep -o '"name":"[^"]*"' | while read -r name; do
    QUEUE_NAME=$(echo "$name" | cut -d'"' -f4)
    echo "  $QUEUE_NAME"
  done
fi

# Summary
TOTAL=$((CREATE_SUCCESS + VOTE_SUCCESS + STATUS_SUCCESS))
echo -e "\n${CYAN}EXTREME LOAD TEST COMPLETE - $TOTAL total ops${NC}"
