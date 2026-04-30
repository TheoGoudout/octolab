"""Unit tests for dispatcher.py."""
import os
import sys
import tempfile

import pytest
import yaml

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from dispatcher import (
    GITLAB_TO_GITHUB_EVENT,
    generate_pipeline,
    get_triggers,
    normalize_triggers,
    sanitize_job_name,
    should_run,
)

FIXTURES = os.path.join(os.path.dirname(__file__), "fixtures")


# --------------------------------------------------------------------------- #
# normalize_triggers
# --------------------------------------------------------------------------- #

def test_normalize_str():
    assert normalize_triggers("push") == {"push": {}}

def test_normalize_list():
    result = normalize_triggers(["push", "pull_request"])
    assert result == {"push": {}, "pull_request": {}}

def test_normalize_dict():
    result = normalize_triggers({"push": {"branches": ["main"]}, "pull_request": None})
    assert result["push"] == {"branches": ["main"]}
    assert result["pull_request"] == {}

def test_normalize_none():
    assert normalize_triggers(None) == {}


# --------------------------------------------------------------------------- #
# get_triggers — PyYAML boolean key handling
# --------------------------------------------------------------------------- #

def test_get_triggers_boolean_key():
    """PyYAML 5.x parses `on:` as True."""
    data = {True: ["push", "pull_request"]}
    assert "push" in get_triggers(data)
    assert "pull_request" in get_triggers(data)

def test_get_triggers_string_key():
    data = {"on": "push"}
    assert get_triggers(data) == {"push": {}}


# --------------------------------------------------------------------------- #
# should_run — push events
# --------------------------------------------------------------------------- #

PUSH_ENV = {"CI_COMMIT_REF_NAME": "main", "CI_COMMIT_TAG": ""}
TAG_ENV  = {"CI_COMMIT_REF_NAME": "v1.0.0", "CI_COMMIT_TAG": "v1.0.0"}
PR_ENV   = {
    "CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "main",
    "CI_MERGE_REQUEST_IID": "42",
}

def test_push_no_filter():
    triggers = {"push": {}}
    assert should_run("push", triggers, PUSH_ENV) is True

def test_push_branch_match():
    triggers = {"push": {"branches": ["main", "develop"]}}
    assert should_run("push", triggers, PUSH_ENV) is True

def test_push_branch_no_match():
    triggers = {"push": {"branches": ["feature/*"]}}
    assert should_run("push", triggers, PUSH_ENV) is False

def test_push_branches_ignore():
    triggers = {"push": {"branches-ignore": ["main"]}}
    assert should_run("push", triggers, PUSH_ENV) is False

def test_push_tag_match():
    triggers = {"push": {"tags": ["v*"]}}
    assert should_run("push", triggers, TAG_ENV) is True

def test_push_tag_no_match():
    triggers = {"push": {"tags": ["release/*"]}}
    assert should_run("push", triggers, TAG_ENV) is False

def test_push_tag_workflow_skipped_on_branch_push():
    """A workflow with only tags: should not run on a branch push."""
    triggers = {"push": {"tags": ["v*"]}}
    assert should_run("push", triggers, PUSH_ENV) is False

def test_push_branch_workflow_skipped_on_tag_push():
    """A workflow with only branches: should not run on a tag push."""
    triggers = {"push": {"branches": ["main"]}}
    assert should_run("push", triggers, TAG_ENV) is False

def test_push_missing_event():
    triggers = {"pull_request": {}}
    assert should_run("push", triggers, PUSH_ENV) is False


# --------------------------------------------------------------------------- #
# should_run — pull_request events
# --------------------------------------------------------------------------- #

def test_pr_no_filter():
    triggers = {"pull_request": {}}
    assert should_run("pull_request", triggers, PR_ENV) is True

def test_pr_branch_match():
    triggers = {"pull_request": {"branches": ["main"]}}
    assert should_run("pull_request", triggers, PR_ENV) is True

def test_pr_branch_no_match():
    triggers = {"pull_request": {"branches": ["staging"]}}
    assert should_run("pull_request", triggers, PR_ENV) is False

def test_pr_branches_ignore():
    triggers = {"pull_request": {"branches-ignore": ["main"]}}
    assert should_run("pull_request", triggers, PR_ENV) is False


# --------------------------------------------------------------------------- #
# should_run — other events
# --------------------------------------------------------------------------- #

def test_workflow_dispatch_matches():
    triggers = {"workflow_dispatch": {}}
    assert should_run("workflow_dispatch", triggers, {}) is True

def test_schedule_matches():
    triggers = {"schedule": [{"cron": "0 2 * * *"}]}
    assert should_run("schedule", triggers, {}) is True

def test_missing_event_returns_false():
    assert should_run("release", {}, {}) is False


# --------------------------------------------------------------------------- #
# sanitize_job_name
# --------------------------------------------------------------------------- #

def test_sanitize_normal():
    assert sanitize_job_name("ci") == "ci"

def test_sanitize_dots_and_spaces():
    assert sanitize_job_name("my workflow.yml") == "my_workflow_yml"

def test_sanitize_leading_digit():
    assert sanitize_job_name("1ci") == "w_1ci"


# --------------------------------------------------------------------------- #
# generate_pipeline
# --------------------------------------------------------------------------- #

def test_generate_pipeline_empty():
    env = {"CI_PIPELINE_SOURCE": "push"}
    pipeline = generate_pipeline([], env)
    assert "bridge_no_workflows" in pipeline
    assert pipeline["stages"] == ["bridge"]

def test_generate_pipeline_single_job():
    env = {"CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "main", "CI_COMMIT_TAG": ""}
    pipeline = generate_pipeline([(".github/workflows/ci.yml", "push")], env)
    assert "bridge_ci" in pipeline
    job = pipeline["bridge_ci"]
    assert job["script"] == ["/entrypoint/bridge.sh"]
    assert job["variables"]["BRIDGE_WORKFLOW_FILE"] == ".github/workflows/ci.yml"
    assert job["variables"]["BRIDGE_EVENT_NAME"] == "push"

def test_generate_pipeline_tag_sets_flag():
    env = {"CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "v1.0.0", "CI_COMMIT_TAG": "v1.0.0"}
    pipeline = generate_pipeline([(".github/workflows/release.yml", "push")], env)
    job = pipeline["bridge_release"]
    assert job["variables"]["BRIDGE_IS_TAG"] == "1"

def test_pipeline_yaml_is_valid():
    """Generated YAML must be loadable."""
    env = {"CI_PIPELINE_SOURCE": "push", "CI_COMMIT_REF_NAME": "main", "CI_COMMIT_TAG": ""}
    pipeline = generate_pipeline([(".github/workflows/ci.yml", "push")], env)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".yml", delete=False) as f:
        yaml.dump(pipeline, f)
        fname = f.name
    with open(fname) as f:
        reloaded = yaml.safe_load(f)
    assert reloaded["stages"] == ["bridge"]


# --------------------------------------------------------------------------- #
# Event mapping
# --------------------------------------------------------------------------- #

def test_gitlab_event_mapping_push():
    assert GITLAB_TO_GITHUB_EVENT["push"] == "push"

def test_gitlab_event_mapping_mr():
    assert GITLAB_TO_GITHUB_EVENT["merge_request_event"] == "pull_request"

def test_gitlab_event_mapping_web():
    assert GITLAB_TO_GITHUB_EVENT["web"] == "workflow_dispatch"


# --------------------------------------------------------------------------- #
# Fixture file parsing
# --------------------------------------------------------------------------- #

@pytest.mark.parametrize("fname,expected_event", [
    ("push_workflow.yml", "push"),
    ("pr_only_workflow.yml", "pull_request"),
    ("tag_workflow.yml", "push"),
    ("dispatch_workflow.yml", "workflow_dispatch"),
    ("schedule_workflow.yml", "schedule"),
])
def test_fixture_triggers(fname, expected_event):
    fpath = os.path.join(FIXTURES, fname)
    with open(fpath) as f:
        data = yaml.safe_load(f)
    triggers = get_triggers(data)
    assert expected_event in triggers, f"expected '{expected_event}' in {triggers}"
