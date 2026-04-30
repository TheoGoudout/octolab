#!/usr/bin/env python3
"""
Generate a GitHub Actions event payload JSON from GitLab CI env vars.
Writes the payload to stdout; bridge.sh redirects it to /tmp/github_event.json.
"""
import json
import os
import sys

env = os.environ


def _repo():
    return {
        "id": int(env.get("CI_PROJECT_ID", "0") or 0),
        "full_name": env.get("CI_PROJECT_PATH", ""),
        "name": env.get("CI_PROJECT_NAME", ""),
        "default_branch": env.get("CI_DEFAULT_BRANCH", "main"),
        "clone_url": env.get("CI_REPOSITORY_URL", ""),
        "html_url": env.get("CI_PROJECT_URL", ""),
        "private": True,
    }


def _sender():
    return {
        "login": env.get("GITLAB_USER_LOGIN", env.get("CI_COMMIT_AUTHOR", "bridge-bot")),
        "id": 0,
    }


def build_push_event() -> dict:
    is_tag = bool(env.get("CI_COMMIT_TAG", ""))
    ref = (
        f"refs/tags/{env['CI_COMMIT_TAG']}"
        if is_tag
        else f"refs/heads/{env.get('CI_COMMIT_REF_NAME', 'main')}"
    )
    return {
        "ref": ref,
        "before": env.get("CI_COMMIT_BEFORE_SHA", "0" * 40),
        "after": env.get("CI_COMMIT_SHA", ""),
        "repository": _repo(),
        "pusher": {"name": env.get("GITLAB_USER_LOGIN", "bridge-bot")},
        "sender": _sender(),
        "commits": [
            {
                "id": env.get("CI_COMMIT_SHA", ""),
                "message": env.get("CI_COMMIT_MESSAGE", ""),
                "author": {
                    "name": env.get("CI_COMMIT_AUTHOR", ""),
                    "email": "",
                },
            }
        ],
        "head_commit": {
            "id": env.get("CI_COMMIT_SHA", ""),
            "message": env.get("CI_COMMIT_MESSAGE", ""),
        },
    }


def build_pull_request_event() -> dict:
    iid = int(env.get("CI_MERGE_REQUEST_IID", "0") or 0)
    return {
        "action": "synchronize",
        "number": iid,
        "pull_request": {
            "number": iid,
            "title": env.get("CI_MERGE_REQUEST_TITLE", ""),
            "state": "open",
            "draft": False,
            "head": {
                "ref": env.get("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", ""),
                "sha": env.get("CI_COMMIT_SHA", ""),
                "repo": _repo(),
            },
            "base": {
                "ref": env.get("CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "main"),
                "repo": _repo(),
            },
            "user": _sender(),
            "body": env.get("CI_MERGE_REQUEST_DESCRIPTION", ""),
        },
        "repository": _repo(),
        "sender": _sender(),
    }


def build_workflow_dispatch_event() -> dict:
    return {
        "inputs": {},
        "ref": f"refs/heads/{env.get('CI_COMMIT_REF_NAME', 'main')}",
        "repository": _repo(),
        "sender": _sender(),
    }


def build_schedule_event() -> dict:
    return {
        "schedule": env.get("CI_PIPELINE_SCHEDULE_DESCRIPTION", ""),
        "repository": _repo(),
        "sender": _sender(),
    }


def build_repository_dispatch_event() -> dict:
    return {
        "action": "bridge",
        "client_payload": {},
        "repository": _repo(),
        "sender": _sender(),
    }


def build_workflow_run_event() -> dict:
    return {
        "action": "completed",
        "workflow_run": {
            "id": int(env.get("CI_PIPELINE_ID", "0") or 0),
            "status": "completed",
            "conclusion": "success",
        },
        "repository": _repo(),
        "sender": _sender(),
    }


BUILDERS = {
    "push": build_push_event,
    "pull_request": build_pull_request_event,
    "workflow_dispatch": build_workflow_dispatch_event,
    "schedule": build_schedule_event,
    "repository_dispatch": build_repository_dispatch_event,
    "workflow_run": build_workflow_run_event,
}


def main():
    event_name = env.get("BRIDGE_EVENT_NAME", "push")
    builder = BUILDERS.get(event_name, build_push_event)
    payload = builder()
    json.dump(payload, sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
