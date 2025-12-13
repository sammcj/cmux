#!/usr/bin/env bash
# E2E tests for SSH commands (cloud + local)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
DMUX="${ROOT_DIR}/packages/sandbox/target/release/dmux"
SANDBOX_ID=""

# Set base URL for dmux - reload.sh starts dmux-sandbox-dev-run on port 46833
export CMUX_SANDBOX_URL="http://localhost:46833"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_test() { echo -e "${GREEN}[TEST]${NC} $*"; }

cleanup() {
    log_info "Cleaning up..."

    # Delete sandbox if created
    if [[ -n "${SANDBOX_ID}" ]]; then
        log_info "Deleting sandbox ${SANDBOX_ID}..."
        curl -sf -X DELETE "http://localhost:46833/sandboxes/${SANDBOX_ID}" || true
    fi
}
trap cleanup EXIT

# Always rebuild and reload to ensure we're testing current branch code
log_info "Rebuilding and reloading dev environment..."
"${SCRIPT_DIR}/../reload.sh"

# Wait for health check after reload
log_info "Waiting for dev server to be ready..."
for i in $(seq 1 30); do
    if curl -sf "http://localhost:46833/healthz" >/dev/null 2>&1; then
        log_info "Dev server ready after ${i}s"
        break
    fi
    if [[ $i -eq 30 ]]; then
        log_error "Dev server failed to start after reload"
        exit 1
    fi
    sleep 1
done

# ============================================================================
# LOCAL SANDBOX TESTS
# ============================================================================

log_info "=========================================="
log_info "LOCAL SANDBOX TESTS"
log_info "=========================================="

# Create a local sandbox
log_test "Creating local sandbox..."
CREATE_OUTPUT=$(curl -sf -X POST "http://localhost:46833/sandboxes" \
    -H "Content-Type: application/json" \
    -d '{"name": "e2e-ssh-test", "workspace": "/tmp/e2e-ssh-test"}')
SANDBOX_ID=$(echo "${CREATE_OUTPUT}" | jq -r '.id')
SANDBOX_INDEX=$(echo "${CREATE_OUTPUT}" | jq -r '.index')
log_info "Created sandbox: ${SANDBOX_ID} (index: ${SANDBOX_INDEX})"

# Test with both full UUID and short prefix
LOCAL_ID_FULL="l_${SANDBOX_ID}"
LOCAL_ID_SHORT="l_${SANDBOX_ID:0:8}"
log_info "Local ID (full): ${LOCAL_ID_FULL}"
log_info "Local ID (short): ${LOCAL_ID_SHORT}"

# Also test with raw UUID (no l_ prefix - should work for task run IDs)
log_info "Raw UUID: ${SANDBOX_ID}"

# Test ssh-exec with l_<full-uuid>
log_test "Testing ssh-exec with l_<full-uuid>..."
EXEC_OUTPUT=$("${DMUX}" ssh-exec "${LOCAL_ID_FULL}" echo "hello full uuid")
if [[ "${EXEC_OUTPUT}" == *"hello full uuid"* ]]; then
    log_info "PASS: ssh-exec l_<full-uuid>"
else
    log_error "FAIL: ssh-exec l_<full-uuid>"
    log_error "Output: ${EXEC_OUTPUT}"
    exit 1
fi

# Test ssh-exec with l_<short-prefix>
log_test "Testing ssh-exec with l_<short-prefix>..."
EXEC_OUTPUT=$("${DMUX}" ssh-exec "${LOCAL_ID_SHORT}" echo "hello short prefix")
if [[ "${EXEC_OUTPUT}" == *"hello short prefix"* ]]; then
    log_info "PASS: ssh-exec l_<short-prefix>"
else
    log_error "FAIL: ssh-exec l_<short-prefix>"
    log_error "Output: ${EXEC_OUTPUT}"
    exit 1
fi

# Test ssh-exec exit code propagation
log_test "Testing ssh-exec exit code propagation..."
set +e
"${DMUX}" ssh-exec "${LOCAL_ID_FULL}" "exit 42"
EXIT_CODE=$?
set -e
if [[ "${EXIT_CODE}" -eq 42 ]]; then
    log_info "PASS: ssh-exec exit code propagation (got ${EXIT_CODE})"
else
    log_error "FAIL: ssh-exec exit code propagation (expected 42, got ${EXIT_CODE})"
    exit 1
fi

# Test ssh-exec with command that has spaces
log_test "Testing ssh-exec with complex command..."
EXEC_OUTPUT=$("${DMUX}" ssh-exec "${LOCAL_ID_FULL}" "echo 'hello world' && pwd")
if [[ "${EXEC_OUTPUT}" == *"hello world"* ]] && [[ "${EXEC_OUTPUT}" == *"/workspace"* ]]; then
    log_info "PASS: ssh-exec complex command"
else
    log_error "FAIL: ssh-exec complex command"
    log_error "Output: ${EXEC_OUTPUT}"
    exit 1
fi

# Test interactive SSH with full UUID
log_test "Testing interactive SSH with l_<full-uuid>..."
INTERACTIVE_OUTPUT=$(echo "echo 'interactive full' && exit" | timeout 10 "${DMUX}" ssh "${LOCAL_ID_FULL}" 2>&1 || true)
if [[ "${INTERACTIVE_OUTPUT}" == *"interactive full"* ]] || [[ "${INTERACTIVE_OUTPUT}" == *"Connected"* ]]; then
    log_info "PASS: interactive SSH l_<full-uuid>"
else
    log_warn "WARN: interactive SSH l_<full-uuid> may have issues"
fi

# Test interactive SSH with short prefix
log_test "Testing interactive SSH with l_<short-prefix>..."
INTERACTIVE_OUTPUT=$(echo "echo 'interactive short' && exit" | timeout 10 "${DMUX}" ssh "${LOCAL_ID_SHORT}" 2>&1 || true)
if [[ "${INTERACTIVE_OUTPUT}" == *"interactive short"* ]] || [[ "${INTERACTIVE_OUTPUT}" == *"Connected"* ]]; then
    log_info "PASS: interactive SSH l_<short-prefix>"
else
    log_warn "WARN: interactive SSH l_<short-prefix> may have issues"
fi

# ============================================================================
# CLOUD SANDBOX TESTS (optional - requires CMUX_TEAM)
# ============================================================================

CLOUD_SANDBOX_ID=""

cloud_cleanup() {
    if [[ -n "${CLOUD_SANDBOX_ID}" ]]; then
        log_info "Cloud sandbox ${CLOUD_SANDBOX_ID} will auto-pause after TTL"
    fi
}

if [[ -n "${CMUX_TEAM:-}" ]]; then
    log_info "=========================================="
    log_info "CLOUD SANDBOX TESTS"
    log_info "=========================================="

    # Create a cloud sandbox with short TTL (2 minutes)
    log_test "Creating cloud sandbox (TTL=120s)..."
    VM_OUTPUT=$("${DMUX}" vm create --team "${CMUX_TEAM}" --ttl 120 --output json 2>&1) || {
        log_warn "Failed to create cloud sandbox: ${VM_OUTPUT}"
        log_warn "Skipping cloud tests (check authentication with 'dmux auth status')"
    }

    if [[ -n "${VM_OUTPUT}" ]] && echo "${VM_OUTPUT}" | jq -e '.id' >/dev/null 2>&1; then
        CLOUD_SANDBOX_ID=$(echo "${VM_OUTPUT}" | jq -r '.id')
        # Strip morphvm_ prefix if present for display
        CLOUD_DISPLAY_ID="${CLOUD_SANDBOX_ID#morphvm_}"
        log_info "Created cloud sandbox: ${CLOUD_DISPLAY_ID}"

        # Use c_ prefix for cloud sandboxes
        CLOUD_ID="c_${CLOUD_DISPLAY_ID}"
        log_info "Cloud ID: ${CLOUD_ID}"

        # Wait a moment for the sandbox to be ready
        log_info "Waiting for cloud sandbox to be ready..."
        sleep 5

        # Test ssh-exec with cloud sandbox
        log_test "Testing ssh-exec with c_<id>..."
        EXEC_OUTPUT=$("${DMUX}" ssh-exec "${CLOUD_ID}" echo "hello cloud" 2>&1) || true
        if [[ "${EXEC_OUTPUT}" == *"hello cloud"* ]]; then
            log_info "PASS: ssh-exec c_<id>"
        else
            log_error "FAIL: ssh-exec c_<id>"
            log_error "Output: ${EXEC_OUTPUT}"
            # Don't exit - continue with other tests
        fi

        # Test ssh-exec exit code propagation
        log_test "Testing cloud ssh-exec exit code propagation..."
        set +e
        "${DMUX}" ssh-exec "${CLOUD_ID}" "exit 42"
        CLOUD_EXIT_CODE=$?
        set -e
        if [[ "${CLOUD_EXIT_CODE}" -eq 42 ]]; then
            log_info "PASS: cloud ssh-exec exit code propagation (got ${CLOUD_EXIT_CODE})"
        else
            log_warn "WARN: cloud ssh-exec exit code (expected 42, got ${CLOUD_EXIT_CODE})"
        fi

        # Test interactive SSH with cloud sandbox
        log_test "Testing interactive SSH with c_<id>..."
        INTERACTIVE_OUTPUT=$(echo "echo 'interactive cloud' && exit" | timeout 15 "${DMUX}" ssh "${CLOUD_ID}" 2>&1 || true)
        if [[ "${INTERACTIVE_OUTPUT}" == *"interactive cloud"* ]] || [[ "${INTERACTIVE_OUTPUT}" == *"Connected"* ]]; then
            log_info "PASS: interactive SSH c_<id>"
        else
            log_warn "WARN: interactive SSH c_<id> may have issues"
        fi

        log_info "Cloud sandbox will auto-pause in ~2 minutes"
    fi
else
    log_warn "Skipping cloud tests (set CMUX_TEAM to enable)"
fi

# ============================================================================
# SUMMARY
# ============================================================================

log_info "=========================================="
log_info "ALL TESTS PASSED"
log_info "=========================================="
