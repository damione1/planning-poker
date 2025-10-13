#!/bin/bash

# WebSocket Stress Test Script
# Tests WebSocket server capacity with multiple rooms and connections

set -e

# Configuration
HOST="${HOST:-localhost:8090}"
ROOMS="${ROOMS:-10}"
CLIENTS_PER_ROOM="${CLIENTS_PER_ROOM:-5}"
DURATION="${DURATION:-30}"
TEMP_FILE="/tmp/stress_test_rooms_$$.txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== WebSocket Stress Test ===${NC}"
echo "Host: $HOST"
echo "Rooms: $ROOMS"
echo "Clients per room: $CLIENTS_PER_ROOM"
echo "Total connections: $((ROOMS * CLIENTS_PER_ROOM))"
echo "Duration: ${DURATION}s"
echo ""

# Check if server is running
echo -e "${YELLOW}Checking server status...${NC}"
if ! curl -s -f http://$HOST/monitoring/health > /dev/null; then
    echo -e "${RED}Error: Server not responding at http://$HOST${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Check for websocat
if ! command -v websocat &> /dev/null; then
    echo -e "${RED}Error: websocat not found${NC}"
    echo "Install with: brew install websocat (macOS) or download from https://github.com/vi/websocat/releases"
    exit 1
fi

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    # Kill all websocat processes
    pkill -f "websocat.*ws://$HOST" 2>/dev/null || true
    # Remove temp file
    rm -f "$TEMP_FILE"
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}
trap cleanup EXIT INT TERM

# Get initial metrics
echo -e "${YELLOW}Initial server metrics:${NC}"
curl -s http://$HOST/monitoring/metrics | jq '{
    active_connections,
    active_rooms,
    memory_usage_mb,
    num_goroutines,
    health_status
}'
echo ""

# Create rooms
echo -e "${YELLOW}Creating $ROOMS test rooms...${NC}"
for i in $(seq 1 $ROOMS); do
    # Create room via HTTP POST and extract room ID from Location header
    LOCATION=$(curl -s -i -X POST http://$HOST/room \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "name=StressTest$i&customValues=1,2,3,5,8,13&pointingMethod=custom" \
        | grep -i '^Location:' | cut -d' ' -f2 | tr -d '\r\n')

    # Extract room ID from /room/{id} path
    ROOM_ID=$(echo "$LOCATION" | sed 's|.*/room/||')

    if [ -z "$ROOM_ID" ]; then
        echo -e "${RED}Failed to create room $i${NC}"
        continue
    fi

    echo "$ROOM_ID" >> "$TEMP_FILE"
    echo -e "${GREEN}✓${NC} Created room $i: $ROOM_ID"
done

CREATED_ROOMS=$(wc -l < "$TEMP_FILE" 2>/dev/null | tr -d ' ')
if [ -z "$CREATED_ROOMS" ]; then
    CREATED_ROOMS=0
fi
echo -e "${GREEN}✓ Created $CREATED_ROOMS rooms${NC}"
echo ""

# Start WebSocket clients
echo -e "${YELLOW}Starting WebSocket clients...${NC}"
CLIENT_COUNT=0

while IFS= read -r ROOM_ID; do
    for j in $(seq 1 $CLIENTS_PER_ROOM); do
        # Start websocat in background
        # Send periodic vote messages
        (
            while true; do
                echo '{"type":"vote","payload":{"value":"'$((RANDOM % 13 + 1))'"}}'
                sleep $((RANDOM % 5 + 3))  # Random 3-8 second intervals
            done | websocat "ws://$HOST/ws/$ROOM_ID" > /dev/null 2>&1
        ) &

        CLIENT_COUNT=$((CLIENT_COUNT + 1))

        # Progress indicator every 10 clients
        if [ $((CLIENT_COUNT % 10)) -eq 0 ]; then
            echo -ne "\rStarted $CLIENT_COUNT/$((ROOMS * CLIENTS_PER_ROOM)) clients..."
        fi

        # Small delay to avoid overwhelming the server during connection phase
        sleep 0.1
    done
done < "$TEMP_FILE"

echo -e "\n${GREEN}✓ Started $CLIENT_COUNT clients${NC}"
echo ""

# Wait for connections to establish
echo -e "${YELLOW}Waiting 5 seconds for connections to establish...${NC}"
sleep 5

# Monitor metrics during test
echo -e "${YELLOW}Monitoring metrics for ${DURATION} seconds...${NC}"
echo ""

for i in $(seq 1 $DURATION); do
    # Get metrics
    METRICS=$(curl -s http://$HOST/monitoring/metrics)

    ACTIVE_CONNS=$(echo "$METRICS" | jq -r '.active_connections')
    ACTIVE_ROOMS=$(echo "$METRICS" | jq -r '.active_rooms')
    MSG_PER_SEC=$(echo "$METRICS" | jq -r '.messages_per_second')
    MEMORY_MB=$(echo "$METRICS" | jq -r '.memory_usage_mb')
    HEALTH=$(echo "$METRICS" | jq -r '.health_status')
    GOROUTINES=$(echo "$METRICS" | jq -r '.num_goroutines')

    # Color code health status
    if [ "$HEALTH" = "healthy" ]; then
        HEALTH_COLOR=$GREEN
    elif [ "$HEALTH" = "warning" ]; then
        HEALTH_COLOR=$YELLOW
    else
        HEALTH_COLOR=$RED
    fi

    # Print metrics
    echo -ne "\r[$i/${DURATION}s] Conns: $ACTIVE_CONNS | Rooms: $ACTIVE_ROOMS | Msg/s: $(printf "%.2f" "$MSG_PER_SEC") | Mem: ${MEMORY_MB}MB | Goroutines: $GOROUTINES | Health: ${HEALTH_COLOR}${HEALTH}${NC}   "

    sleep 1
done

echo ""
echo ""

# Final metrics
echo -e "${YELLOW}Final server metrics:${NC}"
curl -s http://$HOST/monitoring/metrics | jq '{
    active_connections,
    total_connections,
    active_rooms,
    messages_received,
    messages_sent,
    messages_per_second,
    connection_errors,
    broadcast_errors,
    rate_limit_violations,
    memory_usage_mb,
    num_goroutines,
    health_status,
    uptime_seconds
}'

echo ""
echo -e "${GREEN}=== Stress Test Complete ===${NC}"
