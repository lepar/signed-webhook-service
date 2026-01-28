#!/bin/bash

# Test script for webhook HMAC validation
# Usage: ./test_webhooks.sh [base_url] [secret]
# Example: ./test_webhooks.sh http://localhost:8080 my-secret-key

BASE_URL="${1:-http://localhost:8080}"
SECRET="${2:-default-secret-key-change-in-production}"

echo "=========================================="
echo "Webhook HMAC Validation Test Script"
echo "=========================================="
echo "Base URL: $BASE_URL"
echo "Secret: $SECRET"
echo ""

# Helper function to compute HMAC signature
compute_signature() {
    local timestamp=$1
    local nonce=$2
    local body=$3
    local message="${timestamp}\n${nonce}\n${body}"
    echo -ne "$message" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2
}

# Generate test data
TIMESTAMP=$(date +%s)
NONCE1=$(uuidgen | tr '[:upper:]' '[:lower:]')
NONCE2=$(uuidgen | tr '[:upper:]' '[:lower:]')
NONCE3=$(uuidgen | tr '[:upper:]' '[:lower:]')

PAYLOAD='{"user":"testuser","asset":"BTC","amount":"100.5"}'
PAYLOAD_INVALID='{"user":"testuser","asset":"BTC"}'

echo "=========================================="
echo "1. VALID REQUEST (should succeed)"
echo "=========================================="
SIG1=$(compute_signature "$TIMESTAMP" "$NONCE1" "$PAYLOAD")
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE1' \\"
echo "  -H 'X-Signature: $SIG1' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE1" \
  -H "X-Signature: $SIG1" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "2. INVALID SIGNATURE (should fail)"
echo "=========================================="
WRONG_SIG="deadbeefcafebabe1234567890abcdef1234567890abcdef1234567890abcdef12"
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE2' \\"
echo "  -H 'X-Signature: $WRONG_SIG' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE2" \
  -H "X-Signature: $WRONG_SIG" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "3. MISSING X-SIGNATURE HEADER (should fail)"
echo "=========================================="
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE3' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE3" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "4. MISSING X-TIMESTAMP HEADER (should fail)"
echo "=========================================="
NONCE4=$(uuidgen | tr '[:upper:]' '[:lower:]')
SIG4=$(compute_signature "$TIMESTAMP" "$NONCE4" "$PAYLOAD")
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Nonce: $NONCE4' \\"
echo "  -H 'X-Signature: $SIG4' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Nonce: $NONCE4" \
  -H "X-Signature: $SIG4" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "5. MISSING X-NONCE HEADER (should fail)"
echo "=========================================="
SIG5=$(compute_signature "$TIMESTAMP" "" "$PAYLOAD")
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Signature: $SIG5' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Signature: $SIG5" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "6. REPLAY ATTACK (duplicate nonce, should fail)"
echo "=========================================="
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE1' \\"
echo "  -H 'X-Signature: $SIG1' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE1" \
  -H "X-Signature: $SIG1" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "7. EXPIRED TIMESTAMP (should fail)"
echo "=========================================="
OLD_TIMESTAMP=$((TIMESTAMP - 600))  # 10 minutes ago
NONCE7=$(uuidgen | tr '[:upper:]' '[:lower:]')
SIG7=$(compute_signature "$OLD_TIMESTAMP" "$NONCE7" "$PAYLOAD")
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $OLD_TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE7' \\"
echo "  -H 'X-Signature: $SIG7' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $OLD_TIMESTAMP" \
  -H "X-Nonce: $NONCE7" \
  -H "X-Signature: $SIG7" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "8. INVALID JSON BODY (should fail)"
echo "=========================================="
INVALID_BODY='{"user":"testuser","asset":"BTC"}'  # Missing amount
NONCE8=$(uuidgen | tr '[:upper:]' '[:lower:]')
SIG8=$(compute_signature "$TIMESTAMP" "$NONCE8" "$INVALID_BODY")
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE8' \\"
echo "  -H 'X-Signature: $SIG8' \\"
echo "  -d '$INVALID_BODY'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE8" \
  -H "X-Signature: $SIG8" \
  -d "$INVALID_BODY"
echo ""
echo ""

echo "=========================================="
echo "9. WRONG SECRET KEY (should fail)"
echo "=========================================="
# Compute signature with wrong secret
WRONG_SECRET="wrong-secret-key"
NONCE9=$(uuidgen | tr '[:upper:]' '[:lower:]')
WRONG_MESSAGE="${TIMESTAMP}\n${NONCE9}\n${PAYLOAD}"
WRONG_SIG9=$(echo -ne "$WRONG_MESSAGE" | openssl dgst -sha256 -hmac "$WRONG_SECRET" | cut -d' ' -f2)
echo "curl -X POST $BASE_URL/webhook \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -H 'X-Timestamp: $TIMESTAMP' \\"
echo "  -H 'X-Nonce: $NONCE9' \\"
echo "  -H 'X-Signature: $WRONG_SIG9' \\"
echo "  -d '$PAYLOAD'"
echo ""
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $TIMESTAMP" \
  -H "X-Nonce: $NONCE9" \
  -H "X-Signature: $WRONG_SIG9" \
  -d "$PAYLOAD"
echo ""
echo ""

echo "=========================================="
echo "10. GET BALANCE - Existing User (should succeed)"
echo "=========================================="
echo "First, let's add a balance entry via webhook, then query it"
TEST_USER="balance-test-user"
TEST_ASSET="BTC"
TEST_AMOUNT="50.25"
BALANCE_PAYLOAD="{\"user\":\"$TEST_USER\",\"asset\":\"$TEST_ASSET\",\"amount\":\"$TEST_AMOUNT\"}"
BALANCE_TIMESTAMP=$(date +%s)
BALANCE_NONCE=$(uuidgen | tr '[:upper:]' '[:lower:]')
BALANCE_SIG=$(compute_signature "$BALANCE_TIMESTAMP" "$BALANCE_NONCE" "$BALANCE_PAYLOAD")

echo "Adding balance entry via webhook..."
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $BALANCE_TIMESTAMP" \
  -H "X-Nonce: $BALANCE_NONCE" \
  -H "X-Signature: $BALANCE_SIG" \
  -d "$BALANCE_PAYLOAD" \
  -s -o /dev/null -w "Status: %{http_code}\n"
echo ""

echo "Now querying balance for user: $TEST_USER"
echo "curl -X GET $BASE_URL/balance/$TEST_USER"
echo ""
curl -X GET "$BASE_URL/balance/$TEST_USER"
echo ""
echo ""

echo "=========================================="
echo "11. GET BALANCE - Non-existent User (should succeed with empty balances)"
echo "=========================================="
NONEXISTENT_USER="nonexistent-user-$(date +%s)"
echo "curl -X GET $BASE_URL/balance/$NONEXISTENT_USER"
echo ""
curl -X GET "$BASE_URL/balance/$NONEXISTENT_USER"
echo ""
echo ""

echo "=========================================="
echo "12. GET BALANCE - Missing User Parameter (should fail)"
echo "=========================================="
echo "curl -X GET $BASE_URL/balance/"
echo ""
curl -X GET "$BASE_URL/balance/"
echo ""
echo ""

echo "=========================================="
echo "13. GET BALANCE - Multiple Assets for Same User"
echo "=========================================="
MULTI_USER="multi-asset-user"
# Add BTC balance
BTC_TIMESTAMP=$(date +%s)
BTC_NONCE=$(uuidgen | tr '[:upper:]' '[:lower:]')
BTC_PAYLOAD="{\"user\":\"$MULTI_USER\",\"asset\":\"BTC\",\"amount\":\"100.0\"}"
BTC_SIG=$(compute_signature "$BTC_TIMESTAMP" "$BTC_NONCE" "$BTC_PAYLOAD")
echo "Adding BTC balance..."
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $BTC_TIMESTAMP" \
  -H "X-Nonce: $BTC_NONCE" \
  -H "X-Signature: $BTC_SIG" \
  -d "$BTC_PAYLOAD" \
  -s -o /dev/null -w "Status: %{http_code}\n"

# Add ETH balance
sleep 1  # Small delay to ensure different timestamps
ETH_TIMESTAMP=$(date +%s)
ETH_NONCE=$(uuidgen | tr '[:upper:]' '[:lower:]')
ETH_PAYLOAD="{\"user\":\"$MULTI_USER\",\"asset\":\"ETH\",\"amount\":\"250.75\"}"
ETH_SIG=$(compute_signature "$ETH_TIMESTAMP" "$ETH_NONCE" "$ETH_PAYLOAD")
echo "Adding ETH balance..."
curl -X POST "$BASE_URL/webhook" \
  -H "Content-Type: application/json" \
  -H "X-Timestamp: $ETH_TIMESTAMP" \
  -H "X-Nonce: $ETH_NONCE" \
  -H "X-Signature: $ETH_SIG" \
  -d "$ETH_PAYLOAD" \
  -s -o /dev/null -w "Status: %{http_code}\n"
echo ""

echo "Querying balance for user with multiple assets: $MULTI_USER"
echo "curl -X GET $BASE_URL/balance/$MULTI_USER"
echo ""
curl -X GET "$BASE_URL/balance/$MULTI_USER"
echo ""
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Webhook Tests (HMAC Validation):"
echo "  1. Valid request: 200 OK"
echo "  2. Invalid signature: 401 Unauthorized"
echo "  3. Missing X-Signature: 401 Unauthorized"
echo "  4. Missing X-Timestamp: 401 Unauthorized"
echo "  5. Missing X-Nonce: 401 Unauthorized"
echo "  6. Replay attack: 401 Unauthorized"
echo "  7. Expired timestamp: 401 Unauthorized"
echo "  8. Invalid JSON: 400 Bad Request or 500 Internal Server Error"
echo "  9. Wrong secret: 401 Unauthorized"
echo ""
echo "Balance Query Tests:"
echo "  10. Get balance for existing user: 200 OK with balance data"
echo "  11. Get balance for non-existent user: 200 OK with empty balances"
echo "  12. Get balance with missing user parameter: 400 Bad Request"
echo "  13. Get balance for user with multiple assets: 200 OK with all balances"
echo ""

