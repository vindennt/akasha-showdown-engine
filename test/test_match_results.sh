#!/bin/bash

# Test script for match result storage via WebSocket queue
BASE_URL=${BASE_URL:-"http://localhost:8282"}
TEST_EMAIL=${TEST_EMAIL:-"test@example.com"}
TEST_PASSWORD=${TEST_PASSWORD:-"password"}

echo "══════════════════════════════════════════════"
echo "  Match Result Storage - End-to-End Test"
echo "══════════════════════════════════════════════"
echo ""

# Test 1: Health check
echo "1. Health check..."
HEALTH=$(curl -s "$BASE_URL/health/ping")
if [[ "$HEALTH" == *"pong"* ]]; then
  echo "✓ Server is running"
else
  echo "✗ Server health check failed"
  exit 1
fi
echo ""

# Test 2: Trigger a match via queue
echo "2. Triggering match via matchmaking queue..."
echo "   - Adding player 500 to queue..."
curl -s -X POST "$BASE_URL/ws/queue/join" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 500}' > /dev/null

sleep 0.5

echo "   - Adding player 600 to queue..."
curl -s -X POST "$BASE_URL/ws/queue/join" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 600}' > /dev/null

echo "   ✓ Match should be triggered"
echo ""

# Wait for match to complete and result to be stored
echo "3. Waiting for match result to be stored..."
sleep 2
echo "   ✓ Wait complete"
echo ""

# Test 3: Check server logs (manual verification)
echo "══════════════════════════════════════════════"
echo "✓ TEST COMPLETE"
echo "══════════════════════════════════════════════"
echo ""
echo "Check server logs for:"
echo "  - Match result message (e.g., 'User 500 wins against User 600')"
echo "  - Success message (e.g., '[SUCCESS] Match result stored: title=match_result...')"
echo ""
echo "If you see both messages in the server logs, the test PASSED!"
echo ""
