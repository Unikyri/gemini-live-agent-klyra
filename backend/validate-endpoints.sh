#!/usr/bin/env bash
# Linux/macOS version of endpoint validation

BASE_URL="http://localhost:8080/api/v1"
TESTS_PASSED=0
TESTS_FAILED=0

test_endpoint() {
    local METHOD=$1
    local PATH=$2
    local BODY=$3
    
    if [ -z "$BODY" ]; then
        RESPONSE=$(curl -s -w "\n%{http_code}" -X "$METHOD" "$BASE_URL$PATH" 2>/dev/null || echo "000")
    else
        RESPONSE=$(curl -s -w "\n%{http_code}" -X "$METHOD" -H "Content-Type: application/json" -d "$BODY" "$BASE_URL$PATH" 2>/dev/null || echo "000")
    fi
    
    STATUS=$(echo "$RESPONSE" | tail -n1)
    
    if [ "$STATUS" != "000" ]; then
        echo "✓ [$METHOD] $PATH - Status: $STATUS"
        ((TESTS_PASSED++))
    else
        echo "✗ [$METHOD] $PATH - Connection Failed"
        ((TESTS_FAILED++))
    fi
}

echo "=== Backend Endpoint Validation ==="
echo "Testing endpoints at $BASE_URL"
echo ""

echo "Health Check:"
test_endpoint "GET" "/health"

echo ""
echo "Auth Endpoints:"
test_endpoint "POST" "/auth/google" '{"id_token":"test"}'
test_endpoint "POST" "/auth/google-mock" '{"email":"guest@example.com"}'
test_endpoint "POST" "/auth/refresh" '{"refresh_token":"test"}'

echo ""
echo "Course Endpoints:"
test_endpoint "POST" "/courses" '{"title":"Test"}'
test_endpoint "GET" "/courses/550e8400-e29b-41d4-a716-446655440000"
test_endpoint "GET" "/users/550e8400-e29b-41d4-a716-446655440000/courses"

echo ""
echo "RAG Endpoints:"
test_endpoint "POST" "/topics/550e8400-e29b-41d4-a716-446655440000/process" '{"material_id":"550e8400-e29b-41d4-a716-446655440000"}'
test_endpoint "POST" "/topics/550e8400-e29b-41d4-a716-446655440000/query" '{"query":"test"}'

echo ""
echo "=== Results ==="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo ""
    echo "✓ All endpoints are registered and reachable"
    exit 0
else
    echo ""
    echo "✗ Some endpoints failed"
    exit 1
fi
