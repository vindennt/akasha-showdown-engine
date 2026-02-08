#!/bin/bash

# Diagnostic test to troubleshoot Supabase items table
BASE_URL=${BASE_URL:-"http://localhost:8282"}
TEST_EMAIL=${TEST_EMAIL:-"test@example.com"}
TEST_PASSWORD=${TEST_PASSWORD:-"password"}

echo "Supabase Items Table Diagnostic"
echo "================================"
echo ""

# Login
echo "1. Logging in..."
SIGNIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$TEST_EMAIL"'",
    "password": "'"$TEST_PASSWORD"'"
  }')

TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.session.access_token' 2>/dev/null)
USER_ID=$(echo "$SIGNIN_RESPONSE" | jq -r '.session.user.id' 2>/dev/null)

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
  echo "✗ Login failed"
  exit 1
fi

echo "✓ Logged in as $USER_ID"
echo ""

# Try to list existing items
echo "2. Attempting to list items..."
LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/item/get-items" \
  -H "Authorization: Bearer $TOKEN")

echo "Response: $LIST_RESPONSE"
echo ""

# Try creating with minimal data
echo "3. Attempting to create item with title only..."
CREATE1=$(curl -s -X POST "$BASE_URL/item/create-item" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title": "test_minimal"}')

echo "Response: $CREATE1"
echo ""

# Try creating with title and description
echo "4. Attempting to create item with title + description..."
CREATE2=$(curl -s -X POST "$BASE_URL/item/create-item" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title": "test_full", "description": "test description"}')

echo "Response: $CREATE2"
echo ""

echo "================================"
echo "Diagnostic complete"
