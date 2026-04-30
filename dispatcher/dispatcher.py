#!/usr/bin/env python3
"""
GitLab-to-GitHub CI Bridge Dispatcher.

Scans .github/workflows/, determines which workflows match the current
GitLab CI event, and writes generated-ci.yml for a dynamic child pipeline.
"""
import fnmatch
import logging
import os
import re
import sys

import yaml

logging.basicConfig(
    level=os.environ.get("BRIDGE_LOG_LEVEL", "INFO").upper(),
    format="[BRIDGE dispatcher] %(levelname)s %(message)s",
)
log = logging.getLogger(__name__)

# --------------------------------------------------------------------------- #
# Constants
# --------------------------------------------------------------------------- #

GITLAB_TO_GITHUB_EVENT = {
    "push": "push",
    "merge_request_event": "pull_request",
    "external_pull_request_event": "pull_request",
    "web": "workflow_dispatch",
    "api": "workflow_dispatch",
    "chat": "workflow_dispatch",
    "trigger": "repository_dispatch",
    "pipeline": "workflow_run",
    "parent_pipeline": "workflow_run",
    "schedule": "schedule",
}

# Triggers that are not CI pipeline events and must be skipped
SKIP_TRIGGERS = {"workflow_call"}

WORKFLOW_DIR = os.environ.get("BRIDGE_WORKFLOW_DIR", ".github/workflows")
OUTPUT_FILE = os.environ.get("BRIDGE_OUTPUT_FILE", "generated-ci.yml")
RUNNER_IMAGE = os.environ.get("BRIDGE_RUNNER_IMAGE", "octolab/shim-runner:latest")


# --------------------------------------------------------------------------- #
# Trigger normalization
# --------------------------------------------------------------------------- #

def normalize_triggers(on_value) -> dict:
    """
    Convert any YAML form of the `on:` key into a dict of {event: config}.
    Handles str, list, and dict forms.
    """
    if isinstance(on_value, str):
        return {on_value: {}}
    if isinstance(on_value, list):
        return {event: {} for event in on_value}
    if isinstance(on_value, dict):
        return {k: (v or {}) for k, v in on_value.items()}
    return {}


def get_triggers(data: dict) -> dict:
    """
    Extract and normalize the `on:` triggers from a parsed workflow.
    PyYAML 5.x parses `on:` as the boolean key True (YAML 1.1 spec).
    """
    raw = data.get(True) or data.get("on", {})
    return normalize_triggers(raw)


# --------------------------------------------------------------------------- #
# Trigger matching
# --------------------------------------------------------------------------- #

def _matches_any(value: str, patterns: list) -> bool:
    return any(fnmatch.fnmatch(value, p) for p in patterns)


def should_run(github_event: str, triggers: dict, env: dict) -> bool:
    """
    Return True if the workflow should run for the given github_event
    given the current GitLab CI environment variables in env.
    """
    if github_event not in triggers:
        return False

    cfg = triggers[github_event] or {}

    # Warn about unsupported path filters — treat as "match all"
    if "paths" in cfg or "paths-ignore" in cfg:
        log.warning(
            "paths/paths-ignore filters are not implemented in bridge v1 "
            "— treating as match-all"
        )

    if github_event == "push":
        is_tag = bool(env.get("CI_COMMIT_TAG", ""))
        ref = env.get("CI_COMMIT_TAG", "") if is_tag else env.get("CI_COMMIT_REF_NAME", "")

        if is_tag:
            if "tags" in cfg:
                return _matches_any(ref, cfg["tags"])
            if "branches" in cfg and "tags" not in cfg:
                return False  # tag push ≠ branch push
            return True
        else:
            if "tags" in cfg and "branches" not in cfg:
                return False  # tag-only workflow
            branches_ignore = cfg.get("branches-ignore", [])
            if branches_ignore:
                return not _matches_any(ref, branches_ignore)
            branches = cfg.get("branches", [])
            if branches:
                return _matches_any(ref, branches)
            return True

    if github_event == "pull_request":
        target = env.get("CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "")
        branches_ignore = cfg.get("branches-ignore", [])
        if branches_ignore:
            return not _matches_any(target, branches_ignore)
        branches = cfg.get("branches", [])
        if branches:
            return _matches_any(target, branches)
        return True

    # Simple presence check for other events
    return True


# --------------------------------------------------------------------------- #
# Job name sanitization
# --------------------------------------------------------------------------- #

def sanitize_job_name(name: str) -> str:
    """Convert a workflow filename to a valid GitLab CI job name."""
    name = re.sub(r"[^a-zA-Z0-9_-]", "_", name)
    if name and name[0].isdigit():
        name = "w_" + name
    return name


# --------------------------------------------------------------------------- #
# YAML generation
# --------------------------------------------------------------------------- #

def build_job(workflow_file: str, github_event: str, env: dict) -> dict:
    return {
        "stage": "bridge",
        "image": RUNNER_IMAGE,
        "variables": {
            "BRIDGE_WORKFLOW_FILE": workflow_file,
            "BRIDGE_EVENT_NAME": github_event,
            "BRIDGE_IS_TAG": "1" if env.get("CI_COMMIT_TAG") else "0",
            "BRIDGE_MR_IID": env.get("CI_MERGE_REQUEST_IID", ""),
            "BRIDGE_GITLAB_PAT": "${GITLAB_PAT}",
            "BRIDGE_GITLAB_URL": "${CI_SERVER_URL}",
            "BRIDGE_GITLAB_PROJECT_ID": "${CI_PROJECT_ID}",
            "BRIDGE_GITLAB_PROJECT_PATH": "${CI_PROJECT_PATH}",
        },
        "script": ["/entrypoint/bridge.sh"],
        "services": [{"name": "docker:dind", "alias": "docker"}],
        "artifacts": {
            "when": "always",
            "paths": ["bridge-logs/"],
            "expire_in": "1 day",
        },
        "allow_failure": False,
    }


def build_noop_job(reason: str) -> dict:
    return {
        "stage": "bridge",
        "image": "alpine:3.19",
        "script": [f'echo "[BRIDGE] {reason}"'],
        "allow_failure": False,
    }


def generate_pipeline(active_workflows: list, env: dict) -> dict:
    pipeline = {"stages": ["bridge"], "variables": {"BRIDGE_RUNNER_IMAGE": RUNNER_IMAGE}}

    if not active_workflows:
        event = env.get("CI_PIPELINE_SOURCE", "unknown")
        pipeline["bridge_no_workflows"] = build_noop_job(
            f"No GitHub Actions workflows matched event '{event}'. Skipping."
        )
        return pipeline

    for workflow_file, github_event in active_workflows:
        base = os.path.splitext(os.path.basename(workflow_file))[0]
        job_name = "bridge_" + sanitize_job_name(base)
        pipeline[job_name] = build_job(workflow_file, github_event, env)

    return pipeline


# --------------------------------------------------------------------------- #
# Main
# --------------------------------------------------------------------------- #

def scan_workflows(workflow_dir: str) -> list[tuple[str, dict]]:
    """Return list of (filepath, parsed_data) for all valid workflow YAMLs."""
    results = []
    try:
        entries = sorted(os.listdir(workflow_dir))
    except FileNotFoundError:
        log.warning("workflow directory not found: %s", workflow_dir)
        return results

    for fname in entries:
        if not (fname.endswith(".yml") or fname.endswith(".yaml")):
            continue
        fpath = os.path.join(workflow_dir, fname)
        try:
            with open(fpath) as f:
                data = yaml.safe_load(f)
            if not isinstance(data, dict):
                log.warning("skipping %s: not a valid YAML mapping", fpath)
                continue
            results.append((fpath, data))
        except yaml.YAMLError as exc:
            log.warning("skipping %s: YAML parse error: %s", fpath, exc)
    return results


def main():
    env = os.environ.copy()

    gitlab_event = env.get("CI_PIPELINE_SOURCE", "push")
    github_event = GITLAB_TO_GITHUB_EVENT.get(gitlab_event)

    if github_event is None:
        log.warning(
            "unknown CI_PIPELINE_SOURCE '%s' — defaulting to push event", gitlab_event
        )
        github_event = "push"

    log.info(
        "dispatching: gitlab_event=%s -> github_event=%s", gitlab_event, github_event
    )

    workflows = scan_workflows(WORKFLOW_DIR)
    if not workflows:
        log.info("no workflow files found in %s", WORKFLOW_DIR)

    active = []
    for fpath, data in workflows:
        try:
            triggers = get_triggers(data)
        except Exception as exc:
            log.warning("skipping %s: error reading triggers: %s", fpath, exc)
            continue

        # Skip non-CI triggers silently
        for skip in SKIP_TRIGGERS:
            if skip in triggers:
                log.info("skipping trigger '%s' in %s (not a CI event)", skip, fpath)

        if should_run(github_event, triggers, env):
            log.info("matched: %s (triggers: %s)", fpath, list(triggers.keys()))
            active.append((fpath, github_event))
        else:
            log.info("skipped: %s (no match for event '%s')", fpath, github_event)

    pipeline = generate_pipeline(active, env)

    with open(OUTPUT_FILE, "w") as f:
        yaml.dump(pipeline, f, default_flow_style=False, sort_keys=False)

    log.info(
        "wrote %s with %d active workflow(s)",
        OUTPUT_FILE,
        len(active),
    )


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        log.error("dispatcher failed: %s", exc, exc_info=True)
        sys.exit(1)
