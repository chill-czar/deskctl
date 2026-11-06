#!/usr/bin/env bash
# Smoke test for deskctl and desktop skill integration
set -uo pipefail

DESKCTL="${HOME}/.local/bin/deskctl"
if [ ! -x "$DESKCTL" ]; then
    DESKCTL="$(which deskctl 2>/dev/null || echo "")"
fi

if [ -z "$DESKCTL" ]; then
    echo "ERROR: deskctl binary not found."
    exit 1
fi

echo "=== Running deskctl Smoke Tests ==="

PASS_COUNT=0
FAIL_COUNT=0

assert_ok() {
    local test_name="$1"
    local output="$2"
    if echo "$output" | grep -q '"ok": true' || echo "$output" | grep -q '"ok":true'; then
        echo "  [PASS] $test_name"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "  [FAIL] $test_name"
        echo "         Output: $output"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

assert_blocked() {
    local test_name="$1"
    local output="$2"
    if echo "$output" | grep -qi 'violation' || echo "$output" | grep -qi 'blocked' || echo "$output" | grep -qi 'protected zone'; then
        echo "  [PASS] $test_name (correctly blocked)"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo "  [FAIL] $test_name (expected blocked, got: $output)"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# 1. Screen detection
echo "1. Checking dynamic screen discovery..."
OUT=$("$DESKCTL" screens)
assert_ok "deskctl screens" "$OUT"

# 2. Mouse position
echo "2. Checking mouse position..."
OUT=$("$DESKCTL" mouse-position)
assert_ok "deskctl mouse-position" "$OUT"

# 3. Screenshot
echo "3. Checking screenshot capture..."
OUT=$("$DESKCTL" screenshot --screen active --max-width 1280)
assert_ok "deskctl screenshot" "$OUT"

# 4. Windows list
echo "4. Checking window list..."
OUT=$("$DESKCTL" windows)
assert_ok "deskctl windows" "$OUT"

# 5. Clipboard get/set
echo "5. Checking clipboard..."
OUT=$("$DESKCTL" clipboard set "deskctl-smoke-test-token")
assert_ok "deskctl clipboard set" "$OUT"
OUT=$("$DESKCTL" clipboard get)
assert_ok "deskctl clipboard get" "$OUT"

# 6. Scope protection: ~/.ssh access
echo "6. Checking protected zone blocking..."
OUT=$("$DESKCTL" file read "${HOME}/.ssh/id_rsa" 2>/dev/null || true)
assert_blocked "deskctl file read ~/.ssh/id_rsa" "$OUT"

OUT=$("$DESKCTL" run "cat ~/.ssh/id_rsa" 2>/dev/null || true)
assert_blocked "deskctl run 'cat ~/.ssh/id_rsa'" "$OUT"

echo ""
echo "=== Smoke Test Summary ==="
echo "Passed: $PASS_COUNT"
echo "Failed: $FAIL_COUNT"

if [ "$FAIL_COUNT" -eq 0 ]; then
    echo "All smoke tests PASSED!"
    exit 0
else
    echo "Some tests failed!"
    exit 1
fi
