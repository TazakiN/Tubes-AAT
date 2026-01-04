#!/bin/bash
################################################################################
#                    COMPREHENSIVE TEST SCRIPT
#                    CityConnect RabbitMQ Implementation
################################################################################

set -e

API_BASE="http://localhost:8080/api/v1"
REPORT_SERVICE="http://localhost:3002"
NOTIFICATION_SERVICE="http://localhost:3003"
RABBITMQ_API="http://localhost:15672/api"
RABBITMQ_CREDS="cityconnect:cityconnect_secret"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

header() {
    echo ""
    echo -e "${CYAN}======================================================================${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}======================================================================${NC}"
}

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    [ -n "$2" ] && echo "       $2"
    ((TESTS_FAILED++))
}

skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1 - $2"
    ((TESTS_SKIPPED++))
}

################################################################################
# TEST 1: SERVICE HEALTH CHECKS
################################################################################
header "TEST 1: SERVICE HEALTH CHECKS"

# Auth Service
if curl -sf "http://localhost:3001/health" > /dev/null 2>&1; then
    pass "Auth Service (port 3001)"
else
    fail "Auth Service (port 3001)"
fi

# Report Service
if curl -sf "$REPORT_SERVICE/health" > /dev/null 2>&1; then
    pass "Report Service (port 3002)"
else
    fail "Report Service (port 3002)"
fi

# Notification Service
if curl -sf "$NOTIFICATION_SERVICE/health" > /dev/null 2>&1; then
    pass "Notification Service (port 3003)"
else
    fail "Notification Service (port 3003)"
fi

# RabbitMQ
if curl -sf -u "$RABBITMQ_CREDS" "$RABBITMQ_API/overview" > /dev/null 2>&1; then
    pass "RabbitMQ (port 5672/15672)"
else
    fail "RabbitMQ (port 5672/15672)"
fi

# Gateway
if curl -sf "http://localhost:8080/health" > /dev/null 2>&1; then
    pass "Gateway/Nginx (port 8080)"
else
    fail "Gateway/Nginx (port 8080)"
fi

################################################################################
# TEST 2: RABBITMQ QUEUE CONFIGURATION
################################################################################
header "TEST 2: RABBITMQ QUEUE CONFIGURATION"

QUEUES=$(curl -sf -u "$RABBITMQ_CREDS" "$RABBITMQ_API/queues" 2>/dev/null)

if [ -n "$QUEUES" ]; then
    # Main Queues
    for q in "queue.status_updates" "queue.report_created" "queue.vote_received"; do
        if echo "$QUEUES" | jq -e ".[] | select(.name == \"$q\")" > /dev/null 2>&1; then
            pass "Main Queue: $q"
            
            # Check DLX
            DLX=$(echo "$QUEUES" | jq -r ".[] | select(.name == \"$q\") | .arguments.\"x-dead-letter-exchange\" // empty")
            if [ "$DLX" = "cityconnect.notifications.dlx" ]; then
                pass "  → DLX configured on $q"
            else
                fail "  → DLX configured on $q" "Expected cityconnect.notifications.dlx"
            fi
        else
            fail "Main Queue: $q"
        fi
    done
    
    # DLQ Queues
    for q in "queue.status_updates.dlq" "queue.report_created.dlq" "queue.vote_received.dlq"; do
        if echo "$QUEUES" | jq -e ".[] | select(.name == \"$q\")" > /dev/null 2>&1; then
            pass "DLQ Queue: $q"
        else
            fail "DLQ Queue: $q"
        fi
    done
    
    # Exchanges
    EXCHANGES=$(curl -sf -u "$RABBITMQ_CREDS" "$RABBITMQ_API/exchanges" 2>/dev/null)
    if echo "$EXCHANGES" | jq -e '.[] | select(.name == "cityconnect.notifications")' > /dev/null 2>&1; then
        pass "Main Exchange: cityconnect.notifications"
    else
        fail "Main Exchange: cityconnect.notifications"
    fi
    
    if echo "$EXCHANGES" | jq -e '.[] | select(.name == "cityconnect.notifications.dlx")' > /dev/null 2>&1; then
        pass "DLX Exchange: cityconnect.notifications.dlx"
    else
        fail "DLX Exchange: cityconnect.notifications.dlx"
    fi
else
    skip "Queue Configuration" "Cannot connect to RabbitMQ API"
fi

################################################################################
# TEST 3: DATABASE / OUTBOX
################################################################################
header "TEST 3: DATABASE / OUTBOX"

OUTBOX_STATS=$(curl -sf "$REPORT_SERVICE/admin/outbox/stats" 2>/dev/null)
if [ -n "$OUTBOX_STATS" ]; then
    pass "Outbox Stats Endpoint"
    echo "       Current stats: $OUTBOX_STATS"
else
    fail "Outbox Stats Endpoint"
fi

################################################################################
# TEST 4: AUTHENTICATION
################################################################################
header "TEST 4: AUTHENTICATION FLOW"

TEST_EMAIL="test_${RANDOM}@test.com"
TEST_PASSWORD="password123"

# Register
REG_RESPONSE=$(curl -sf -X POST "$API_BASE/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"name\":\"Test User\",\"role\":\"warga\"}" 2>/dev/null)

if [ -n "$REG_RESPONSE" ]; then
    USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user.id // empty')
    if [ -n "$USER_ID" ]; then
        pass "User Registration"
    else
        fail "User Registration" "No user ID in response"
    fi
else
    fail "User Registration"
fi

# Login
LOGIN_RESPONSE=$(curl -sf -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" 2>/dev/null)

if [ -n "$LOGIN_RESPONSE" ]; then
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty')
    if [ -n "$TOKEN" ]; then
        pass "User Login"
    else
        fail "User Login" "No token in response"
    fi
else
    fail "User Login"
fi

################################################################################
# TEST 5: MESSAGE FLOW - REPORT CREATION
################################################################################
header "TEST 5: MESSAGE FLOW - REPORT CREATION"

if [ -n "$TOKEN" ] && [ -n "$USER_ID" ]; then
    # Get initial outbox stats
    BEFORE_STATS=$(curl -sf "$REPORT_SERVICE/admin/outbox/stats" 2>/dev/null | jq -r '.outbox_stats.published // 0')
    
    # Create report
    REPORT_RESPONSE=$(curl -sf -X POST "$REPORT_SERVICE/" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $USER_ID" \
        -d "{\"title\":\"Test Report $(date +%H%M%S)\",\"description\":\"Testing\",\"category_id\":1,\"privacy_level\":\"public\"}" 2>/dev/null)
    
    if [ -n "$REPORT_RESPONSE" ]; then
        REPORT_ID=$(echo "$REPORT_RESPONSE" | jq -r '.report.id // empty')
        if [ -n "$REPORT_ID" ]; then
            pass "Create Report"
            
            sleep 2
            
            AFTER_STATS=$(curl -sf "$REPORT_SERVICE/admin/outbox/stats" 2>/dev/null | jq -r '.outbox_stats.published // 0')
            if [ "$AFTER_STATS" -gt "$BEFORE_STATS" ]; then
                pass "Outbox Published Message"
            else
                fail "Outbox Published Message" "Before: $BEFORE_STATS, After: $AFTER_STATS"
            fi
        else
            fail "Create Report" "No report ID"
        fi
    else
        fail "Create Report"
    fi
else
    skip "Report Creation" "No auth token"
fi

################################################################################
# TEST 6: VOTE
################################################################################
header "TEST 6: MESSAGE FLOW - VOTE"

if [ -n "$TOKEN" ] && [ -n "$USER_ID" ] && [ -n "$REPORT_ID" ]; then
    VOTE_RESPONSE=$(curl -sf -X POST "$REPORT_SERVICE/$REPORT_ID/vote" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $USER_ID" \
        -d '{"vote_type":"upvote"}' 2>/dev/null)
    
    if [ -n "$VOTE_RESPONSE" ]; then
        pass "Cast Vote"
    else
        fail "Cast Vote"
    fi
else
    skip "Vote" "Missing token or report ID"
fi

################################################################################
# TEST 7: DLQ CHECK
################################################################################
header "TEST 7: DLQ STATUS"

if [ -n "$QUEUES" ]; then
    DLQ_TOTAL=$(echo "$QUEUES" | jq '[.[] | select(.name | test("\\.dlq$")) | .messages] | add // 0')
    
    if [ "$DLQ_TOTAL" -eq 0 ]; then
        pass "DLQ Empty (no failed messages)"
    else
        fail "DLQ has $DLQ_TOTAL messages" "Check RabbitMQ UI for details"
    fi
else
    skip "DLQ Check" "Cannot connect to RabbitMQ"
fi

################################################################################
# SUMMARY
################################################################################
header "TEST SUMMARY"

TOTAL=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
if [ $TOTAL -gt 0 ]; then
    PASS_RATE=$((TESTS_PASSED * 100 / TOTAL))
else
    PASS_RATE=0
fi

echo ""
echo "  Total Tests:  $TOTAL"
echo -e "  ${GREEN}Passed:       $TESTS_PASSED${NC}"
echo -e "  ${RED}Failed:       $TESTS_FAILED${NC}"
echo -e "  ${YELLOW}Skipped:      $TESTS_SKIPPED${NC}"
echo ""
echo "  Pass Rate:    $PASS_RATE%"
echo ""

if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Some tests failed! Check output above.${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
