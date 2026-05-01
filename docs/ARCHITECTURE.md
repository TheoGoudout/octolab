# Architecture

This document describes the internal design of the octolab CI bridge.

## Overview

The bridge is built around two independent stages that communicate through a GitLab dynamic child pipeline artifact.

```
┌──────────────────────────────────────────────────────────────────────────┐
│  GitLab Parent Pipeline                                                  │
│                                                                          │
│  .pre stage                          bridge-execute stage                │
│  ┌─────────────────────┐             ┌──────────────────────────────┐   │
│  │  bridge-dispatch    │──artifact──▶│  trigger: generated-ci.yml   │   │
│  │  (dispatcher.py)    │             └──────────┬───────────────────┘   │
│  └─────────────────────┘                        │                        │
└─────────────────────────────────────────────────│────────────────────────┘
                                                  │  child pipeline
                                                  ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  GitLab Child Pipeline (generated-ci.yml)                                │
│                                                                          │
│  bridge stage                                                            │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  bridge_ci_yml  (shim-runner image)                             │    │
│  │                                                                  │    │
│  │  ┌─────────────┐   env vars   ┌───────────┐    GitHub API      │    │
│  │  │  bridge.sh  │─────────────▶│   act     │◀──────────────┐    │    │
│  │  │  (entrypoint│              │ (nektos)  │               │    │    │
│  │  └─────────────┘              └─────┬─────┘    ┌──────────┴──┐ │    │
│  │                                     │          │  octoshim   │ │    │
│  │                                     │          │  proxy :8080│ │    │
│  │                                     ▼          └──────────┬──┘ │    │
│  │                            .github/workflows/ci.yml       │    │    │
│  │                                                            │    │    │
│  │  services: docker:dind ◀────────────────────────────────  │    │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────────────┘
```

## Stage A: The Dispatcher

**File:** `dispatcher/dispatcher.py`
**Image:** `octolab/bridge-dispatcher`
**Stage:** `.pre` (always runs first)

The dispatcher is a stateless Python script. It reads from environment variables and the filesystem, writes one artifact, and exits.

### Input

| Source | Variable / Path | Purpose |
|--------|----------------|---------|
| Filesystem | `.github/workflows/*.yml` | Workflow files to evaluate |
| GitLab CI env | `CI_PIPELINE_SOURCE` | Maps to a GitHub event name |
| GitLab CI env | `CI_COMMIT_REF_NAME` | Branch/tag name for push trigger filtering |
| GitLab CI env | `CI_COMMIT_TAG` | Set when the push is a tag |
| GitLab CI env | `CI_MERGE_REQUEST_TARGET_BRANCH_NAME` | Target branch for PR trigger filtering |

### Processing

1. **Event mapping** — `CI_PIPELINE_SOURCE` is translated to a GitHub event name using a fixed lookup table (e.g. `merge_request_event` → `pull_request`).

2. **Workflow scanning** — All `.yml`/`.yaml` files in `BRIDGE_WORKFLOW_DIR` are parsed with PyYAML. Malformed files are skipped with a warning.

3. **Trigger extraction** — The `on:` key is read from each workflow. PyYAML 5.x parses `on:` as the boolean key `True` (YAML 1.1 spec); the dispatcher handles both `True` and `"on"` as valid keys.

4. **Trigger matching** — For each workflow, `should_run()` evaluates whether the current event and branch/tag satisfy the workflow's trigger filters:
   - `push.branches` / `push.branches-ignore` — glob-matched against `CI_COMMIT_REF_NAME`
   - `push.tags` / `push.tags-ignore` — glob-matched against `CI_COMMIT_TAG`
   - `pull_request.branches` / `pull_request.branches-ignore` — matched against the MR target branch
   - `paths` / `paths-ignore` — **not implemented in v1**; treated as match-all with a log warning
   - `workflow_call` — silently skipped (not a CI pipeline event trigger)

5. **Child pipeline generation** — One GitLab CI job is emitted per matched workflow. If no workflows match, a no-op `echo` job is emitted so the pipeline does not fail.

### Output

`generated-ci.yml` — a valid GitLab CI YAML file defining the child pipeline. Example:

```yaml
stages:
  - bridge

bridge_ci_yml:
  stage: bridge
  image: octolab/shim-runner:latest
  variables:
    BRIDGE_WORKFLOW_FILE: .github/workflows/ci.yml
    BRIDGE_EVENT_NAME: push
    BRIDGE_IS_TAG: "0"
    BRIDGE_MR_IID: ""
    BRIDGE_GITLAB_PAT: ${GITLAB_PAT}
    BRIDGE_GITLAB_URL: ${CI_SERVER_URL}
    BRIDGE_GITLAB_PROJECT_ID: ${CI_PROJECT_ID}
    BRIDGE_GITLAB_PROJECT_PATH: ${CI_PROJECT_PATH}
  script:
    - /entrypoint/bridge.sh
  services:
    - name: docker:dind
      alias: docker
  artifacts:
    when: always
    paths:
      - bridge-logs/
    expire_in: 1 day
  allow_failure: false
```

## Stage B: The Executor

**Image:** `octolab/shim-runner`
**Entrypoint:** `/entrypoint/bridge.sh`

Each child pipeline job runs one GitHub Actions workflow inside the shim image. The job has three cooperating processes:

### bridge.sh (entrypoint)

Seven sequential phases:

| Phase | Action |
|-------|--------|
| 1 | Source `mask-tokens.sh`: validate `GITLAB_PAT`, emit `::add-mask::` directives |
| 2 | Start `octoshim` in background; poll `/health` until ready (max 5 s) |
| 3 | Export all `GITHUB_*` and `RUNNER_*` variables mapped from `CI_*` variables |
| 4 | Run `generate-event.py` → `/tmp/github_event.json` |
| 5 | Write env file and secrets file (chmod 600) for `act` |
| 6 | Invoke `act`, pipe output through `sed` masking filter, tee to `bridge-logs/` |
| 7 | Kill octoshim, delete temp files, propagate `act` exit code |

### act (nektos/act)

`act` parses the target GitHub Actions YAML file and executes each job and step. It:
- Resolves `uses: actions/checkout@v4` etc. by downloading Marketplace actions
- Handles `$GITHUB_OUTPUT`, `$GITHUB_ENV`, `$GITHUB_PATH` file protocols
- Runs Docker actions via the DinD service (`--container-daemon-socket tcp://docker:2376`)
- Uses the pre-configured `~/.config/act/actrc` to map `ubuntu-latest` to `catthehacker/ubuntu:act-latest`

### octoshim (API proxy)

A lightweight HTTP server on `:8080`. When workflow steps call GitHub API endpoints (e.g. `actions/github-script`), those calls go to `GITHUB_API_URL=http://localhost:8080` and are transparently translated.

See [API-PROXY.md](API-PROXY.md) for the full endpoint reference.

## Environment Variable Mapping

| GitHub Variable | Source | Transformation |
|----------------|--------|---------------|
| `GITHUB_WORKSPACE` | `CI_PROJECT_DIR` | direct |
| `GITHUB_REPOSITORY` | `CI_PROJECT_PATH` | direct |
| `GITHUB_REPOSITORY_NAME` | `CI_PROJECT_NAME` | direct |
| `GITHUB_REPOSITORY_OWNER` | `CI_PROJECT_NAMESPACE` | direct |
| `GITHUB_SHA` | `CI_COMMIT_SHA` | direct |
| `GITHUB_REF_NAME` | `CI_COMMIT_REF_NAME` | direct |
| `GITHUB_REF` | `CI_COMMIT_TAG` / `CI_COMMIT_REF_NAME` | `refs/tags/X` if tag, else `refs/heads/X` |
| `GITHUB_HEAD_REF` | `CI_MERGE_REQUEST_SOURCE_BRANCH_NAME` | MR events only |
| `GITHUB_BASE_REF` | `CI_MERGE_REQUEST_TARGET_BRANCH_NAME` | MR events only |
| `GITHUB_EVENT_NAME` | `BRIDGE_EVENT_NAME` | set by dispatcher |
| `GITHUB_EVENT_PATH` | `/tmp/github_event.json` | generated by `generate-event.py` |
| `GITHUB_RUN_ID` | `CI_PIPELINE_ID` | direct |
| `GITHUB_RUN_NUMBER` | `CI_PIPELINE_IID` | direct |
| `GITHUB_JOB` | `CI_JOB_NAME` | direct |
| `GITHUB_ACTIONS` | constant | `"true"` |
| `GITHUB_API_URL` | constant | `"http://localhost:8080"` |
| `GITHUB_SERVER_URL` | constant | `"http://localhost:8080"` |
| `GITHUB_TOKEN` | generated | random hex placeholder; proxy intercepts auth |
| `GITHUB_OUTPUT` | constant | `/github/output/output.txt` |
| `GITHUB_ENV` | constant | `/github/output/env.txt` |
| `GITHUB_PATH` | constant | `/github/output/path.txt` |
| `GITHUB_STEP_SUMMARY` | constant | `/github/output/step_summary.md` |
| `ACTIONS_RUNTIME_TOKEN` | `CI_JOB_TOKEN` | for artifact action compatibility |
| `RUNNER_NAME` | `CI_RUNNER_ID` | `"gitlab-bridge-{id}"` |
| `RUNNER_WORKSPACE` | `CI_PROJECT_DIR` | direct |
| `RUNNER_TEMP` | constant | `/tmp/runner` |

## Container Image Layers

The `Dockerfile` uses a three-stage build to minimize the final image size:

```
golang:1.22-alpine          alpine:3.19
      │                           │
      │ go build -ldflags="-s -w" │ curl + sha256sum
      │                           │ nektos/act binary
      ▼                           ▼
octoshim (static ~5 MB)     act binary
      │                           │
      └──────────┬────────────────┘
                 ▼
         ubuntu:24.04 (runtime)
         + git, curl, jq, python3, nodejs, docker.io
         + /github/{home,workflow,workspace,artifacts,output}
         + /root/.config/act/actrc
         + /entrypoint/
         + /bridge/dispatcher.py
```

## Security Model

```
act workflow step
    │
    │ Authorization: token <random-placeholder>
    ▼
octoshim proxy
    │ strips Authorization header
    │ injects PRIVATE-TOKEN: <GITLAB_PAT>
    ▼
GitLab API (https://gitlab.com)
```

`GITLAB_PAT` never appears in:
- `act` output (masked by `::add-mask::` directives)
- `bridge-logs/` artifacts (filtered by inline `sed` pipeline)
- The child pipeline YAML artifact (`${GITLAB_PAT}` is a GitLab CI variable reference, not the literal value)
- HTTP logs (octoshim's logger redacts the PAT value from all output)
