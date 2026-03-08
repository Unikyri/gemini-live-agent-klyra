#!/usr/bin/env bash
# Linux/macOS test verification script - equivalent to test-all.ps1

set -e

echo "=== Klyra Backend Test Suite ==="
echo "Testing all core functionality: Auth, Course, Material, RAG, Handlers"
echo ""

cd "$(dirname "$0")/backend"

echo "Running all unit tests..."
go test -v -coverprofile=coverage.out ./...

if [ -f coverage.out ]; then
    echo ""
    echo "Coverage Summary:"
    go tool cover -func=coverage.out | tail -n 1
else
    echo "WARNING: No coverage report generated"
fi

echo ""
echo "✓ All tests completed successfully"
