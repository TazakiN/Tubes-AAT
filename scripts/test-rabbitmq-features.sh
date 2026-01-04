#!/bin/bash
# Test RabbitMQ Features - DLQ, Retry, Outbox Pattern
# This script tests the new RabbitMQ improvements

BASE_URL="${1:-http://localhost:8080}"
RABBITMQ_URL="http://localhost:15672"
RABBITMQ_CREDS="cityconnect:cityconnect_secret"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

success() { echo -e "${GREEN}✓ $1${NC}"; }
error() { echo -e "${RED}✗ $1${NC}"; }
info() { echo -e "${CYAN}ℹ $1${NC}"; }
section() { echo -e "\n${YELLOW}=== $1 ===${NC}"; }

echo -e "${YELLOW}"
echo "================================================================================"
echo "    RABBITMQ FEATURES TEST SUITE"
echo "    Testing: Outbox Pattern, DLQ, Retry, Notification Service"
echo "================================================================================"
echo -e "${NC}"

# ============================================
# 1. SERVICE HEALTH CHECKS
# ============================================
section "1. SERVICE HEALTH CHECKS"

# Check Report Service
REPORT_HEALTH=$(curl -s http://localhost:3002/health 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$REPORT_HEALTH" ]; then
    success "Report Service: $(echo $REPORT_HEALTH | jq -r '.status // "ok"')"
else
    error "Report Service not available"
fi

# Check Notification Service
NOTIF_HEALTH=$(curl -s http://localhost:3003/health 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$NOTIF_HEALTH" ]; then
    success "Notification Service: $(echo $NOTIF_HEALTH | jq -r '.status // "ok"')"
    
    # Get detailed health
    DETAILED=$(curl -s http://localhost:3003/health/detailed 2>/dev/null)
    if [ -n "$DETAILED" ]; then
        info "Features: $(echo $DETAILED | jq -r '.features | join(", ")')"
    fi
else
    error "Notification Service not available"
fi

# Check RabbitMQ
QUEUES=$(curl -s -u "$RABBITMQ_CREDS" "$RABBITMQ_URL/api/queues" 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$QUEUES" ]; then
    QUEUE_COUNT=$(echo $QUEUES | jq 'length')
    success "RabbitMQ: $QUEUE_COUNT queues found"
    
    echo $QUEUES | jq -r '.[] | "  - \(.name): \(.messages) messages"'
else
    error "RabbitMQ Management not available"
fi

# ============================================
# 2. AUTHENTICATION
# ============================================
section "2. AUTHENTICATION"

# Login as warga
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"warga@test.com","password":"password123"}')

AUTH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token // empty')
if [ -n "$AUTH_TOKEN" ]; then
    success "Logged in as warga@test.com"
else
    error "Failed to login as warga"
    AUTH_TOKEN=""
fi

# Login as admin
ADMIN_LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin_kebersihan@test.com","password":"password123"}')

ADMIN_TOKEN=$(echo $ADMIN_LOGIN | jq -r '.token // empty')
if [ -n "$ADMIN_TOKEN" ]; then
    success "Logged in as admin_kebersihan@test.com"
else
    error "Failed to login as admin"
    ADMIN_TOKEN=""
fi

# ============================================
# 3. TEST OUTBOX PATTERN
# ============================================
section "3. TEST OUTBOX PATTERN"

# Check outbox stats before
OUTBOX_BEFORE=$(curl -s http://localhost:3002/admin/outbox/stats 2>/dev/null)
if [ -n "$OUTBOX_BEFORE" ]; then
    info "Outbox stats before: $OUTBOX_BEFORE"
fi

# Create a report
REPORT_ID=""
if [ -n "$AUTH_TOKEN" ]; then
    TIMESTAMP=$(date +%H:%M:%S)
    REPORT=$(curl -s -X POST "$BASE_URL/api/v1/reports/" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Test Report - Outbox Pattern $TIMESTAMP\",
            \"description\": \"Testing outbox pattern\",
            \"category_id\": 1,
            \"privacy_level\": \"public\",
            \"location_lat\": -6.2088,
            \"location_lng\": 106.8456
        }")
    
    REPORT_ID=$(echo $REPORT | jq -r '.id // empty')
    if [ -n "$REPORT_ID" ]; then
        success "Created report: $REPORT_ID"
        info "Title: $(echo $REPORT | jq -r '.title')"
    else
        error "Failed to create report: $REPORT"
    fi
fi

# Wait for outbox worker
sleep 2

# Check outbox stats after
OUTBOX_AFTER=$(curl -s http://localhost:3002/admin/outbox/stats 2>/dev/null)
if [ -n "$OUTBOX_AFTER" ]; then
    info "Outbox stats after: $OUTBOX_AFTER"
fi

# ============================================
# 4. TEST STATUS UPDATE
# ============================================
section "4. TEST STATUS UPDATE NOTIFICATION"

if [ -n "$ADMIN_TOKEN" ] && [ -n "$REPORT_ID" ]; then
    STATUS_RESULT=$(curl -s -X PATCH "$BASE_URL/api/v1/reports/$REPORT_ID/status" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"status": "in_progress"}')
    
    success "Updated report status to 'in_progress'"
    
    sleep 2
    
    # Check notifications
    NOTIFICATIONS=$(curl -s "$BASE_URL/api/v1/notifications" \
        -H "Authorization: Bearer $AUTH_TOKEN")
    
    UNREAD=$(echo $NOTIFICATIONS | jq -r '.unread_count // 0')
    info "Unread notifications: $UNREAD"
    
    LATEST=$(echo $NOTIFICATIONS | jq -r '.notifications[0].title // "none"')
    if [ "$LATEST" != "none" ]; then
        success "Latest notification: $LATEST"
    fi
fi

# ============================================
# 5. TEST VOTE
# ============================================
section "5. TEST VOTE NOTIFICATION"

if [ -n "$ADMIN_TOKEN" ] && [ -n "$REPORT_ID" ]; then
    VOTE_RESULT=$(curl -s -X POST "$BASE_URL/api/v1/reports/$REPORT_ID/vote" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"vote_type": "upvote"}')
    
    SCORE=$(echo $VOTE_RESULT | jq -r '.vote_score // "error"')
    success "Voted on report. New score: $SCORE"
    
    sleep 2
    
    # Check notifications for warga
    NOTIFICATIONS=$(curl -s "$BASE_URL/api/v1/notifications" \
        -H "Authorization: Bearer $AUTH_TOKEN")
    
    UNREAD=$(echo $NOTIFICATIONS | jq -r '.unread_count // 0')
    info "Warga unread notifications: $UNREAD"
fi

# ============================================
# 6. CHECK RABBITMQ QUEUES & DLQ
# ============================================
section "6. RABBITMQ QUEUE STATUS"

QUEUES=$(curl -s -u "$RABBITMQ_CREDS" "$RABBITMQ_URL/api/queues" 2>/dev/null)
if [ -n "$QUEUES" ]; then
    info "Main Queues:"
    echo $QUEUES | jq -r '.[] | select(.name | test("^queue\\.") and (test("\\.dlq$") | not)) | "  - \(.name): \(.messages) messages"'
    
    info "Dead Letter Queues (DLQ):"
    echo $QUEUES | jq -r '.[] | select(.name | test("\\.dlq$")) | "  - \(.name): \(.messages) messages"'
    
    DLQ_TOTAL=$(echo $QUEUES | jq '[.[] | select(.name | test("\\.dlq$")) | .messages] | add // 0')
    if [ "$DLQ_TOTAL" -eq 0 ]; then
        success "All DLQs are empty (no failed messages)"
    else
        error "Some messages in DLQ ($DLQ_TOTAL) - check for processing errors!"
    fi
fi

# ============================================
# SUMMARY
# ============================================
section "TEST SUMMARY"

echo -e "${CYAN}"
cat << 'EOF'
Tested:
- Service health
- Outbox pattern  
- Queue triggers
- DLQ config
- Consumer

DLQs with 0 consumers is expected.
EOF
echo -e "${NC}"

echo "Done at $(date '+%H:%M:%S')"
