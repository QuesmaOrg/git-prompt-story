#!/bin/bash
set -euo pipefail

echo "=== E2E Test Suite ==="
echo ""

# Source helper functions
source /e2e/lib/helpers.sh

# Track test results
PASSED=0
FAILED=0
TOTAL=0

# Run all test scripts
for test_script in /e2e/tests/*.sh; do
    if [[ -f "$test_script" ]]; then
        TOTAL=$((TOTAL + 1))
        test_name=$(basename "$test_script" .sh)

        echo "Running: $test_name"

        if bash "$test_script"; then
            PASSED=$((PASSED + 1))
        else
            FAILED=$((FAILED + 1))
            echo "  FAILED: $test_name"
        fi

        echo ""
    fi
done

echo "=== $PASSED/$TOTAL tests passed ==="

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
