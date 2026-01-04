#!/bin/bash
################################################################################
#                    EXTREME LOAD TEST - 2.5M USERS TARGET
#                    CityConnect RabbitMQ Stress Test
################################################################################

# Default parameters
USERS=${USERS:-1000}
REPORTS_PER_USER=${REPORTS_PER_USER:-10}
VOTES_PER_REPORT=${VOTES_PER_REPORT:-5}
BATCH_SIZE=${BATCH_SIZE:-50}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --users) USERS="$2"; shift 2 ;;
        --reports) REPORTS_PER_USER="$2"; shift 2 ;;
        --votes) VOTES_PER_REPORT="$2"; shift 2 ;;
        --insane) 
            USERS=10000
            REPORTS_PER_USER=25
            VOTES_PER_REPORT=10
            BATCH_SIZE=200
            echo ""
            echo "██╗███╗   ██╗███████╗ █████╗ ███╗   ██╗███████╗    ███╗   ███╗ ██████╗ ██████╗ ███████╗"
            echo "██║████╗  ██║██╔════╝██╔══██╗████╗  ██║██╔════╝    ████╗ ████║██╔═══██╗██╔══██╗██╔════╝"
            echo "██║██╔██╗ ██║███████╗███████║██╔██╗ ██║█████╗      ██╔████╔██║██║   ██║██║  ██║█████╗  "
            echo "██║██║╚██╗██║╚════██║██╔══██║██║╚██╗██║██╔══╝      ██║╚██╔╝██║██║   ██║██║  ██║██╔══╝  "
            echo "██║██║ ╚████║███████║██║  ██║██║ ╚████║███████╗    ██║ ╚═╝ ██║╚██████╔╝██████╔╝███████╗"
            echo "╚═╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝    ╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝"
            echo ""
            shift ;;
        *) shift ;;
    esac
done

API_BASE="http://localhost:8080/api/v1"
REPORT_SERVICE="http://localhost:3002"
RABBITMQ_API="http://localhost:15672/api"
RABBITMQ_CREDS="cityconnect:cityconnect_secret"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# Statistics
USERS_CREATED=0
USERS_FAILED=0
REPORTS_CREATED=0
REPORTS_FAILED=0
VOTES_CAST=0
VOTES_FAILED=0
STATUS_UPDATES=0
STATUS_FAILED=0
TOTAL_REQUESTS=0

banner() {
    echo ""
    echo -e "${1}████████████████████████████████████████████████████████████████████████████████${NC}"
    echo -e "${1}█  $2${NC}"
    echo -e "${1}████████████████████████████████████████████████████████████████████████████████${NC}"
}

progress() {
    local current=$1
    local total=$2
    local text=$3
    local percent=$((current * 100 / total))
    local filled=$((percent / 2))
    local empty=$((50 - filled))
    printf "\r[%${filled}s%${empty}s] %d%% - %s" "" "" "$percent" "$text" | tr ' ' '█' | sed "s/█\{$empty\}$/$(printf '%*s' $empty '' | tr ' ' '░')/"
}

get_rabbitmq_stats() {
    curl -sf -u "$RABBITMQ_CREDS" "$RABBITMQ_API/overview" 2>/dev/null
}

################################################################################
# MAIN
################################################################################

clear
echo ""
echo -e "${MAGENTA}╔═══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${MAGENTA}║                                                                               ║${NC}"
echo -e "${MAGENTA}║   ███████╗██╗  ██╗████████╗██████╗ ███████╗███╗   ███╗███████╗                ║${NC}"
echo -e "${MAGENTA}║   ██╔════╝╚██╗██╔╝╚══██╔══╝██╔══██╗██╔════╝████╗ ████║██╔════╝                ║${NC}"
echo -e "${MAGENTA}║   █████╗   ╚███╔╝    ██║   ██████╔╝█████╗  ██╔████╔██║█████╗                  ║${NC}"
echo -e "${MAGENTA}║   ██╔══╝   ██╔██╗    ██║   ██╔══██╗██╔══╝  ██║╚██╔╝██║██╔══╝                  ║${NC}"
echo -e "${MAGENTA}║   ███████╗██╔╝ ██╗   ██║   ██║  ██║███████╗██║ ╚═╝ ██║███████╗                ║${NC}"
echo -e "${MAGENTA}║   ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝     ╚═╝╚══════╝                ║${NC}"
echo -e "${MAGENTA}║                                                                               ║${NC}"
echo -e "${MAGENTA}║                    LOAD TEST - TARGET 2.5 MILLION USERS                       ║${NC}"
echo -e "${MAGENTA}║                                                                               ║${NC}"
echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

echo -e "${YELLOW}Configuration:${NC}"
echo "  Concurrent Users:    $USERS"
echo "  Reports per User:    $REPORTS_PER_USER"
echo "  Votes per Report:    $VOTES_PER_REPORT"
echo "  Batch Size:          $BATCH_SIZE"
echo ""

TOTAL_REPORTS=$((USERS * REPORTS_PER_USER))
TOTAL_VOTES=$((TOTAL_REPORTS * VOTES_PER_REPORT))
TOTAL_OPS=$((USERS + TOTAL_REPORTS + TOTAL_VOTES))

echo -e "${YELLOW}Expected Operations:${NC}"
echo "  Total Users:         $USERS"
echo "  Total Reports:       $TOTAL_REPORTS"
echo "  Total Votes:         $TOTAL_VOTES"
echo "  Total Operations:    $TOTAL_OPS"
echo ""

# Start timer
START_TIME=$(date +%s.%N)

################################################################################
# PHASE 1: USER REGISTRATION
################################################################################
banner "$GREEN" "PHASE 1: REGISTERING $USERS USERS"

declare -a USER_TOKENS
declare -a USER_IDS

for ((i=1; i<=USERS; i++)); do
    if ((i % 50 == 0 || i == USERS)); then
        printf "\r[%d/%d] Registering users..." "$i" "$USERS"
    fi
    
    EMAIL="loadtest_${i}_${RANDOM}@test.com"
    
    REG_RESPONSE=$(curl -sf -X POST "$API_BASE/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$EMAIL\",\"password\":\"password123\",\"name\":\"User $i\",\"role\":\"warga\"}" 2>/dev/null)
    
    if [ -n "$REG_RESPONSE" ]; then
        LOGIN_RESPONSE=$(curl -sf -X POST "$API_BASE/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"$EMAIL\",\"password\":\"password123\"}" 2>/dev/null)
        
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty' 2>/dev/null)
        USER_ID=$(echo "$REG_RESPONSE" | jq -r '.user.id // empty' 2>/dev/null)
        
        if [ -n "$TOKEN" ] && [ -n "$USER_ID" ]; then
            USER_TOKENS+=("$TOKEN")
            USER_IDS+=("$USER_ID")
            ((USERS_CREATED++))
        else
            ((USERS_FAILED++))
        fi
    else
        ((USERS_FAILED++))
    fi
    
    ((TOTAL_REQUESTS+=2))
done

echo ""
echo -e "${GREEN}Users registered: $USERS_CREATED/$USERS${NC}"

################################################################################
# PHASE 2: REPORT CREATION
################################################################################
banner "$CYAN" "PHASE 2: CREATING $TOTAL_REPORTS REPORTS"

declare -a REPORT_IDS
REPORT_COUNT=0

for idx in "${!USER_TOKENS[@]}"; do
    TOKEN="${USER_TOKENS[$idx]}"
    USER_ID="${USER_IDS[$idx]}"
    
    for ((r=1; r<=REPORTS_PER_USER; r++)); do
        ((REPORT_COUNT++))
        
        if ((REPORT_COUNT % 100 == 0 || REPORT_COUNT == TOTAL_REPORTS)); then
            printf "\r[%d/%d] Creating reports..." "$REPORT_COUNT" "$TOTAL_REPORTS"
        fi
        
        RESPONSE=$(curl -sf -X POST "$REPORT_SERVICE/" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -H "X-User-ID: $USER_ID" \
            -d "{\"title\":\"Report $REPORT_COUNT\",\"description\":\"Load test\",\"category_id\":$((RANDOM % 8 + 1)),\"privacy_level\":\"public\"}" 2>/dev/null)
        
        REPORT_ID=$(echo "$RESPONSE" | jq -r '.report.id // empty' 2>/dev/null)
        
        if [ -n "$REPORT_ID" ]; then
            REPORT_IDS+=("$REPORT_ID")
            ((REPORTS_CREATED++))
        else
            ((REPORTS_FAILED++))
        fi
        
        ((TOTAL_REQUESTS++))
    done
done

echo ""
echo -e "${GREEN}Reports created: $REPORTS_CREATED/$TOTAL_REPORTS${NC}"

################################################################################
# PHASE 3: VOTING
################################################################################
banner "$YELLOW" "PHASE 3: VOTING STORM"

VOTE_COUNT=0
EXPECTED_VOTES=$((${#REPORT_IDS[@]} * VOTES_PER_REPORT))

for REPORT_ID in "${REPORT_IDS[@]}"; do
    for ((v=1; v<=VOTES_PER_REPORT; v++)); do
        ((VOTE_COUNT++))
        
        if ((VOTE_COUNT % 200 == 0 || VOTE_COUNT == EXPECTED_VOTES)); then
            printf "\r[%d/%d] Casting votes..." "$VOTE_COUNT" "$EXPECTED_VOTES"
        fi
        
        # Random voter
        VOTER_IDX=$((RANDOM % ${#USER_TOKENS[@]}))
        TOKEN="${USER_TOKENS[$VOTER_IDX]}"
        USER_ID="${USER_IDS[$VOTER_IDX]}"
        
        VOTE_TYPE=$([ $((RANDOM % 2)) -eq 0 ] && echo "upvote" || echo "downvote")
        
        RESPONSE=$(curl -sf -X POST "$REPORT_SERVICE/$REPORT_ID/vote" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: application/json" \
            -H "X-User-ID: $USER_ID" \
            -d "{\"vote_type\":\"$VOTE_TYPE\"}" 2>/dev/null)
        
        if [ -n "$RESPONSE" ]; then
            ((VOTES_CAST++))
        else
            ((VOTES_FAILED++))
        fi
        
        ((TOTAL_REQUESTS++))
    done
done

echo ""
echo -e "${GREEN}Votes cast: $VOTES_CAST/$EXPECTED_VOTES${NC}"

################################################################################
# RESULTS
################################################################################
END_TIME=$(date +%s.%N)
DURATION=$(echo "$END_TIME - $START_TIME" | bc)
OPS_PER_SEC=$(echo "scale=2; $TOTAL_REQUESTS / $DURATION" | bc)

banner "$MAGENTA" "LOAD TEST COMPLETE"

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                              RESULTS SUMMARY                                  ║${NC}"
echo -e "${CYAN}╠═══════════════════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║  Duration:              $(printf '%-50s' "${DURATION}s")║${NC}"
echo -e "${CYAN}║  Total Requests:        $(printf '%-50s' "$TOTAL_REQUESTS")║${NC}"
echo -e "${CYAN}║  Throughput:            $(printf '%-50s' "${OPS_PER_SEC} req/sec")║${NC}"
echo -e "${CYAN}╠═══════════════════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║  Users Created:         $(printf '%-50s' "$USERS_CREATED")║${NC}"
echo -e "${CYAN}║  Reports Created:       $(printf '%-50s' "$REPORTS_CREATED")║${NC}"
echo -e "${CYAN}║  Votes Cast:            $(printf '%-50s' "$VOTES_CAST")║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# RabbitMQ stats
echo -e "${CYAN}RabbitMQ Status:${NC}"
QUEUES=$(curl -sf -u "$RABBITMQ_CREDS" "$RABBITMQ_API/queues" 2>/dev/null)
if [ -n "$QUEUES" ]; then
    TOTAL_MSG=$(echo "$QUEUES" | jq '[.[].messages] | add // 0')
    DLQ_MSG=$(echo "$QUEUES" | jq '[.[] | select(.name | test("\\.dlq$")) | .messages] | add // 0')
    echo "  Messages in queues: $TOTAL_MSG"
    echo "  Messages in DLQ:    $DLQ_MSG"
fi

# Outbox stats
OUTBOX=$(curl -sf "$REPORT_SERVICE/admin/outbox/stats" 2>/dev/null)
if [ -n "$OUTBOX" ]; then
    echo ""
    echo -e "${CYAN}Outbox Stats:${NC} $OUTBOX"
fi

echo ""
echo -e "${MAGENTA}════════════════════════════════════════════════════════════════════════════════${NC}"
