#!/bin/bash
# Run acceptance tests for the VergeIO Terraform provider
#
# Required Environment Variables:
#   VERGEOS_HOST       - VergeOS API endpoint (e.g., https://192.168.1.138)
#   VERGEOS_USERNAME   - Username for authentication (or use VERGEOS_TOKEN)
#   VERGEOS_PASSWORD   - Password for authentication (or use VERGEOS_TOKEN)
#   VERGEOS_TOKEN      - API token (alternative to username/password)
#
# Optional Environment Variables:
#   VERGEOS_TEST_VNET_ID  - VNet ID for VNet rule tests (required for vnet_rule tests)
#   VERGEOS_TEST_TARGET_IP - Target IP for NAT rule tests (defaults to 192.168.0.100)
#
# Usage:
#   ./scripts/run-tests.sh                     # Run all tests
#   ./scripts/run-tests.sh TestAccTag          # Run tests matching "TestAccTag"
#   ./scripts/run-tests.sh TestAccVNetRule -v  # Run with verbose output

set -e

# Default settings
TIMEOUT="${TIMEOUT:-30m}"
TEST_PATTERN="${1:-}"
VERBOSE="${2:-}"

# Check required environment variables
if [ -z "$VERGEOS_HOST" ]; then
    echo "ERROR: VERGEOS_HOST environment variable must be set"
    echo "Example: export VERGEOS_HOST=https://192.168.1.138"
    exit 1
fi

if [ -z "$VERGEOS_TOKEN" ] && ([ -z "$VERGEOS_USERNAME" ] || [ -z "$VERGEOS_PASSWORD" ]); then
    echo "ERROR: Either VERGEOS_TOKEN or both VERGEOS_USERNAME and VERGEOS_PASSWORD must be set"
    exit 1
fi

echo "================================================"
echo "VergeIO Terraform Provider - Acceptance Tests"
echo "================================================"
echo "Host: $VERGEOS_HOST"
echo "Auth: ${VERGEOS_TOKEN:+Token}${VERGEOS_USERNAME:+Username/Password}"
echo "Pattern: ${TEST_PATTERN:-All tests}"
echo "Timeout: $TIMEOUT"
echo ""

# Build the test command
TEST_CMD="TF_ACC=1 go test ./internal/provider/... -timeout=$TIMEOUT"

if [ -n "$TEST_PATTERN" ]; then
    TEST_CMD="$TEST_CMD -run='$TEST_PATTERN'"
fi

if [ "$VERBOSE" = "-v" ]; then
    TEST_CMD="$TEST_CMD -v"
fi

echo "Running: $TEST_CMD"
echo ""

# Execute tests
eval $TEST_CMD
