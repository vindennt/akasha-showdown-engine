#!/bin/bash

# Test script for authentication endpoints
BASE_URL="http://localhost:8282"

echo "Testing Authentication Endpoints"
echo "================================="
echo ""

# Test 1: Health Check
echo "1. Testing /health/ping..."
curl -s "$BASE_URL/health/ping"
echo -e "\n"

# Test 2: Signup
echo "2. Testing /auth/signup..."
SIGNUP_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/signup" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password"
  }')
echo "$SIGNUP_RESPONSE" | jq '.' 2>/dev/null || echo "$SIGNUP_RESPONSE"
echo ""

# Test 3: Signin
echo "3. Testing /auth/signin..."
SIGNIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/signin" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password"
  }')
echo "$SIGNIN_RESPONSE" | jq '.' 2>/dev/null || echo "$SIGNIN_RESPONSE"

# Extract token if signin was successful
TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.session.access_token' 2>/dev/null)

if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
  echo ""
  echo "✓ Login successful! Token obtained."
  echo ""
  
  # Test 4: Get Items (authenticated)
  echo "4. Testing /item/get-items (authenticated)..."
  curl -s -X GET "$BASE_URL/item/get-items" \
    -H "Authorization: Bearer $TOKEN" | jq '.' 2>/dev/null || echo "Failed"
  echo ""
else
  echo ""
  echo "✗ Login failed. Cannot test authenticated endpoints."
  echo ""
fi
