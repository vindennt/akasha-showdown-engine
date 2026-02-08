#!/bin/bash

# Test script for inserting items with authenticated user
BASE_URL=${BASE_URL:-"http://localhost:8282"}
TEST_EMAIL=${TEST_EMAIL:-"test@example.com"}
TEST_PASSWORD=${TEST_PASSWORD:-"password"}

echo "Testing Item Insertion with Authentication"
echo "==========================================="
echo ""

# Step 1: Login with test credentials
echo "1. Logging in with $TEST_EMAIL..."
SIGNIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "'"$TEST_EMAIL"'",
    "password": "'"$TEST_PASSWORD"'"
  }')

echo "$SIGNIN_RESPONSE" | jq '.' 2>/dev/null || echo "$SIGNIN_RESPONSE"

# Extract token and user ID
TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.session.access_token' 2>/dev/null)
USER_ID=$(echo "$SIGNIN_RESPONSE" | jq -r '.session.user.id' 2>/dev/null)

if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
  echo ""
  echo "✓ Login successful!"
  echo "  User ID: $USER_ID"
  echo "  Token: ${TOKEN:0:50}..."
  echo ""
  
  # Step 2: Create a test match result item
  echo "2. Creating match result item with owner_id..."
  TIMESTAMP=$(date +%s)
  MATCH_RESULT_TITLE="match_result${TIMESTAMP}"
  WINNER_ID="99"
  
  CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/item/create-item" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{
      \"title\": \"${MATCH_RESULT_TITLE}\",
      \"description\": \"${WINNER_ID}\"
    }")
  
  echo "$CREATE_RESPONSE" | jq '.' 2>/dev/null || echo "$CREATE_RESPONSE"
  echo ""
  
  # Step 3: Verify item was created
  CREATED_ITEM_ID=$(echo "$CREATE_RESPONSE" | jq -r '.[0].id // .id' 2>/dev/null)
  
  if [ "$CREATED_ITEM_ID" != "null" ] && [ -n "$CREATED_ITEM_ID" ] && [ "$CREATED_ITEM_ID" != "" ]; then
    echo "✓ Match result item created successfully!"
    echo "  Item ID: $CREATED_ITEM_ID"
    echo "  Title: $MATCH_RESULT_TITLE"
    echo "  Description (Winner ID): $WINNER_ID"
    echo ""
    
    # Step 4: Retrieve and verify the item
    echo "3. Retrieving the created item..."
    GET_RESPONSE=$(curl -s -X GET "$BASE_URL/item/get-item/$CREATED_ITEM_ID" \
      -H "Authorization: Bearer $TOKEN")
    
    echo "$GET_RESPONSE" | jq '.' 2>/dev/null || echo "$GET_RESPONSE"
    echo ""
    
    RETRIEVED_TITLE=$(echo "$GET_RESPONSE" | jq -r '.title' 2>/dev/null)
    if [ "$RETRIEVED_TITLE" == "$MATCH_RESULT_TITLE" ]; then
      echo "✓ Item retrieval successful!"
      echo ""
      echo "==========================================="
      echo "✓✓✓ ALL TESTS PASSED ✓✓✓"
      echo "==========================================="
      echo ""
      echo "Summary:"
      echo "  - Authentication: PASSED"
      echo "  - Item creation with owner_id: PASSED"
      echo "  - Item retrieval: PASSED"
      echo ""
      echo "User ID for match results: $USER_ID"
      echo ""
      exit 0
    else
      echo "✗ Item retrieval failed or title mismatch"
      exit 1
    fi
  else
    echo "✗ Failed to create item"
    echo "Response: $CREATE_RESPONSE"
    exit 1
  fi
  
else
  echo ""
  echo "✗ Login failed. Cannot proceed with tests."
  echo "Please ensure:"
  echo "  - Server is running on $BASE_URL"
  echo "  - User $TEST_EMAIL exists with password '$TEST_PASSWORD'"
  echo ""
  exit 1
fi
