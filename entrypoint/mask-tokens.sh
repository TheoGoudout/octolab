#!/usr/bin/env bash
# Validate secrets, emit act mask directives, and export BRIDGE_* variables.
# Must be sourced (not executed) from bridge.sh.
set -euo pipefail

PAT="${GITLAB_PAT:-${BRIDGE_GITLAB_PAT:-}}"

if [[ -z "$PAT" ]]; then
  echo "[BRIDGE] ERROR: GITLAB_PAT (or BRIDGE_GITLAB_PAT) must be set as a masked CI variable" >&2
  exit 1
fi

# Emit act log masking directives so the PAT never appears in logs
echo "::add-mask::${PAT}"
[[ -n "${CI_JOB_TOKEN:-}" ]] && echo "::add-mask::${CI_JOB_TOKEN}"

# Export canonical names consumed by octoshim and bridge.sh
export BRIDGE_GITLAB_PAT="$PAT"
export BRIDGE_GITLAB_URL="${CI_SERVER_URL:-https://gitlab.com}"
export BRIDGE_GITLAB_PROJECT_ID="${CI_PROJECT_ID:-}"
export BRIDGE_GITLAB_PROJECT_PATH="${CI_PROJECT_PATH:-}"
