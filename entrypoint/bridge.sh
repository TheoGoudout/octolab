#!/usr/bin/env bash
# Bridge entrypoint: maps GitLab CI variables to GitHub Actions env,
# starts octoshim proxy, generates the event payload, then calls act.
set -euo pipefail

LOG_PREFIX="[BRIDGE]"

log() { echo "${LOG_PREFIX} $*"; }
err() { echo "${LOG_PREFIX} ERROR: $*" >&2; }

# --------------------------------------------------------------------------- #
# Phase 1 — Token masking and BRIDGE_* env setup
# --------------------------------------------------------------------------- #
# shellcheck source=entrypoint/mask-tokens.sh
source /entrypoint/mask-tokens.sh

# --------------------------------------------------------------------------- #
# Phase 2 — Start octoshim proxy
# --------------------------------------------------------------------------- #
log "Starting octoshim API proxy on :8080"
octoshim &
OCTOSHIM_PID=$!
trap 'log "Stopping octoshim (pid ${OCTOSHIM_PID})"; kill "${OCTOSHIM_PID}" 2>/dev/null || true' EXIT

# Wait for the proxy to be ready (max 5 s)
READY=0
for _ in $(seq 1 10); do
  if curl -sf http://localhost:8080/health >/dev/null 2>&1; then
    READY=1
    break
  fi
  sleep 0.5
done
if [[ $READY -eq 0 ]]; then
  err "octoshim did not become ready within 5 seconds"
  exit 1
fi
log "octoshim ready"

# --------------------------------------------------------------------------- #
# Phase 3 — Export GitHub Actions environment variables
# --------------------------------------------------------------------------- #
export GITHUB_WORKSPACE="${CI_PROJECT_DIR:-/github/workspace}"
export GITHUB_REPOSITORY="${CI_PROJECT_PATH:-}"
export GITHUB_REPOSITORY_NAME="${CI_PROJECT_NAME:-}"
export GITHUB_REPOSITORY_OWNER="${CI_PROJECT_NAMESPACE:-}"
export GITHUB_SHA="${CI_COMMIT_SHA:-}"
export GITHUB_REF_NAME="${CI_COMMIT_REF_NAME:-}"
export GITHUB_RUN_ID="${CI_PIPELINE_ID:-0}"
export GITHUB_RUN_NUMBER="${CI_PIPELINE_IID:-0}"
export GITHUB_JOB="${CI_JOB_NAME:-}"
export GITHUB_ACTIONS="true"
export GITHUB_API_URL="http://localhost:8080"
export GITHUB_SERVER_URL="http://localhost:8080"
export RUNNER_WORKSPACE="${CI_PROJECT_DIR:-/github/workspace}"
export RUNNER_TEMP="/tmp/runner"
export RUNNER_NAME="gitlab-bridge-${CI_RUNNER_ID:-0}"
export ACTIONS_RUNTIME_TOKEN="${CI_JOB_TOKEN:-}"

# Build GITHUB_REF (refs/tags/X or refs/heads/X)
if [[ -n "${CI_COMMIT_TAG:-}" ]]; then
  export GITHUB_REF="refs/tags/${CI_COMMIT_TAG}"
else
  export GITHUB_REF="refs/heads/${CI_COMMIT_REF_NAME:-main}"
fi

# MR-specific vars (only meaningful on pull_request events)
if [[ -n "${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}" ]]; then
  export GITHUB_HEAD_REF="${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME}"
  export GITHUB_BASE_REF="${CI_MERGE_REQUEST_TARGET_BRANCH_NAME:-main}"
fi

export GITHUB_EVENT_NAME="${BRIDGE_EVENT_NAME:-push}"

# GITHUB_TOKEN is a random placeholder — octoshim intercepts auth headers
export GITHUB_TOKEN
GITHUB_TOKEN="bridge-token-$(openssl rand -hex 8)"

# File protocol paths
mkdir -p /github/output
export GITHUB_OUTPUT="/github/output/output.txt"
export GITHUB_ENV="/github/output/env.txt"
export GITHUB_PATH="/github/output/path.txt"
export GITHUB_STEP_SUMMARY="/github/output/step_summary.md"
touch "$GITHUB_OUTPUT" "$GITHUB_ENV" "$GITHUB_PATH" "$GITHUB_STEP_SUMMARY"

# --------------------------------------------------------------------------- #
# Phase 4 — Generate GitHub event payload
# --------------------------------------------------------------------------- #
log "Generating event payload for event '${BRIDGE_EVENT_NAME:-push}'"
python3 /entrypoint/generate-event.py > /tmp/github_event.json
export GITHUB_EVENT_PATH="/tmp/github_event.json"

# --------------------------------------------------------------------------- #
# Phase 5 — Write act env and secrets files
# --------------------------------------------------------------------------- #
ENV_FILE="/tmp/bridge-github.env"
SECRET_FILE="/tmp/bridge-secrets"

env | grep -E '^(GITHUB_|RUNNER_|ACTIONS_)' | grep -v 'GITHUB_TOKEN' > "$ENV_FILE" || true
printf 'GITHUB_TOKEN=%s\n' "$GITHUB_TOKEN" > "$SECRET_FILE"
chmod 600 "$SECRET_FILE"

# --------------------------------------------------------------------------- #
# Phase 6 — Invoke act
# --------------------------------------------------------------------------- #
mkdir -p bridge-logs/

WORKFLOW_FILE="${BRIDGE_WORKFLOW_FILE:-.github/workflows}"
EVENT_NAME="${BRIDGE_EVENT_NAME:-push}"
DEFAULT_BRANCH="${CI_DEFAULT_BRANCH:-main}"

log "Running: act ${EVENT_NAME} -W ${WORKFLOW_FILE}"

# Inline PAT masking: replace any literal occurrence before writing to log file
act "${EVENT_NAME}" \
    -W "${WORKFLOW_FILE}" \
    -e /tmp/github_event.json \
    --env-file "${ENV_FILE}" \
    --secret-file "${SECRET_FILE}" \
    --artifact-server-path /github/artifacts \
    --container-daemon-socket tcp://docker:2376 \
    --defaultbranch "${DEFAULT_BRANCH}" \
    --rm \
    2>&1 \
  | sed "s/${BRIDGE_GITLAB_PAT}/[MASKED]/g" \
  | tee bridge-logs/act-output.log

ACT_EXIT="${PIPESTATUS[0]}"

# --------------------------------------------------------------------------- #
# Phase 7 — Cleanup
# --------------------------------------------------------------------------- #
rm -f "${SECRET_FILE}" "${ENV_FILE}" /tmp/github_event.json

log "act exited with code ${ACT_EXIT}"
exit "${ACT_EXIT}"
