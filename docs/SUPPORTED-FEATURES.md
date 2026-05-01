# Supported Features

This document lists which GitHub Actions features are supported by the bridge, which are planned, and which cannot be translated.

## Trigger Events

| GitHub `on:` trigger | Bridge support | Notes |
|----------------------|----------------|-------|
| `push` | ✅ Full | Branch push and tag push both supported |
| `push.branches` | ✅ Full | Glob patterns via `fnmatch` |
| `push.branches-ignore` | ✅ Full | Glob patterns via `fnmatch` |
| `push.tags` | ✅ Full | Glob patterns via `fnmatch` |
| `push.tags-ignore` | ✅ Full | Glob patterns via `fnmatch` |
| `push.paths` | ⚠️ Planned v2 | Currently treated as match-all; a warning is logged |
| `push.paths-ignore` | ⚠️ Planned v2 | Currently treated as match-all; a warning is logged |
| `pull_request` | ✅ Full | Mapped from `merge_request_event` |
| `pull_request.branches` | ✅ Full | Matched against MR target branch |
| `pull_request.branches-ignore` | ✅ Full | Matched against MR target branch |
| `pull_request.paths` | ⚠️ Planned v2 | Treated as match-all |
| `pull_request.types` | ⚠️ Partial | Only `synchronize`/`opened` activity types generated |
| `workflow_dispatch` | ✅ Full | Triggered by `web`, `api`, `chat` pipeline sources |
| `workflow_dispatch.inputs` | ⚠️ Partial | Empty inputs object; inputs are not forwarded |
| `schedule` | ✅ Full | Triggered by GitLab scheduled pipelines |
| `repository_dispatch` | ✅ Full | Triggered by `trigger` pipeline source |
| `workflow_run` | ✅ Full | Triggered by `pipeline` / `parent_pipeline` sources |
| `workflow_call` | ❌ Skipped | Reusable workflows are not CI triggers; silently skipped |
| `release` | ❌ Not mapped | No direct GitLab pipeline source equivalent |
| `create` / `delete` | ❌ Not mapped | Not triggered by GitLab CI |
| `issue_comment` | ❌ Not mapped | GitLab CI does not have comment-triggered pipelines |

## Workflow & Job Features

| Feature | Status | Notes |
|---------|--------|-------|
| `jobs.<id>.steps` | ✅ Full | via `act` |
| `jobs.<id>.needs` | ✅ Full | via `act` |
| `jobs.<id>.if` | ✅ Full | via `act` expression engine |
| `jobs.<id>.strategy.matrix` | ✅ Full | via `act` |
| `jobs.<id>.continue-on-error` | ✅ Full | via `act` |
| `jobs.<id>.timeout-minutes` | ✅ Full | via `act` |
| `jobs.<id>.container` | ✅ Full | via DinD |
| `jobs.<id>.services` | ✅ Full | via DinD |
| `jobs.<id>.outputs` | ✅ Full | via `$GITHUB_OUTPUT` |
| `jobs.<id>.environment` | ❌ Not supported | See below |
| `jobs.<id>.concurrency` | ❌ Not supported | No GitLab equivalent |
| `jobs.<id>.permissions` | ⚠️ Ignored | GITHUB_TOKEN is a placeholder |

## Step Features

| Feature | Status | Notes |
|---------|--------|-------|
| `uses:` (Marketplace actions) | ✅ Full | Downloaded by `act` |
| `uses:` local actions (`./`) | ✅ Full | Resolved relative to workspace |
| `uses:` Docker actions | ✅ Full | Run via DinD |
| `run:` shell steps | ✅ Full | bash/sh/python/etc. |
| `env:` step-level env vars | ✅ Full | via `act` |
| `with:` inputs | ✅ Full | via `act` |
| `if:` conditionals | ✅ Full | via `act` expression engine |
| `continue-on-error` | ✅ Full | via `act` |
| `$GITHUB_OUTPUT` | ✅ Full | File pre-created at `/github/output/output.txt` |
| `$GITHUB_ENV` | ✅ Full | File pre-created at `/github/output/env.txt` |
| `$GITHUB_PATH` | ✅ Full | File pre-created at `/github/output/path.txt` |
| `$GITHUB_STEP_SUMMARY` | ✅ Full | File pre-created; content written but not rendered |

## GitHub API (via Octoshim Proxy)

| GitHub API call | Status | Translated to |
|----------------|--------|---------------|
| `POST /repos/.../check-runs` | ✅ | `POST /projects/.../statuses/:sha` |
| `PATCH /repos/.../check-runs/:id` | ✅ | `POST /projects/.../statuses/:sha` |
| `POST /repos/.../issues/:n/comments` | ✅ | `POST /projects/.../merge_requests/:iid/notes` |
| `GET /repos/.../issues/:n/comments` | ✅ | Returns empty array |
| `GET /repos/.../pulls/:n` | ✅ | `GET /projects/.../merge_requests/:iid` |
| `GET /repos/.../pulls` | ✅ | `GET /projects/.../merge_requests` |
| `POST /repos/.../issues` | ✅ | `POST /projects/.../issues` |
| `GET /repos/.../commits/:sha` | ✅ | `GET /projects/.../repository/commits/:sha` |
| `GET /repos/.../labels` | ✅ | `GET /projects/.../labels` |
| `POST /repos/.../releases` | ❌ 501 | No equivalent (logs NOT SUPPORTED) |
| `POST /repos/.../deployments` | ❌ 501 | No equivalent (logs NOT SUPPORTED) |
| `PUT /repos/.../environments/.../secrets` | ❌ 501 | No equivalent (logs NOT SUPPORTED) |
| `POST /graphql` | ❌ 501 | Not translatable |
| Any other endpoint | ❌ 501 | Logs method + path, continues |

When an API call returns 501, the workflow step that made the call will see an error. Other steps and jobs continue normally.

## Common Marketplace Actions

| Action | Status | Notes |
|--------|--------|-------|
| `actions/checkout` | ✅ | Clones into `GITHUB_WORKSPACE` |
| `actions/setup-node` | ✅ | Installs Node; Node is pre-installed in shim image |
| `actions/setup-python` | ✅ | Installs Python; Python is pre-installed |
| `actions/upload-artifact` | ✅ | Written to `/github/artifacts` |
| `actions/download-artifact` | ✅ | Read from `/github/artifacts` |
| `actions/cache` | ⚠️ Partial | Cache is local to the job; cross-job caching via GitLab cache is not wired |
| `actions/github-script` | ✅ | API calls go through Octoshim proxy |
| `docker/build-push-action` | ✅ | Requires DinD; builder runs against Docker socket |
| `goreleaser/goreleaser-action` | ✅ | Runs as a shell step |

## Not Supported

### GitHub Environments

GitHub's deployment environment protection rules (required reviewers, wait timers, branch policies) have no direct GitLab CI equivalent. Workflow steps that POST to `/repos/.../environments` or `/repos/.../deployments` receive HTTP 501. The workflow continues; only that specific API call fails.

**Workaround:** Remove the `environment:` key from jobs before bridging, or use GitLab's native environments.

### `workflow_call` (Reusable Workflows)

Workflows triggered only by `on: workflow_call` are skipped by the dispatcher because `workflow_call` is not a CI pipeline source — it is only valid when called from another workflow. A log notice is emitted.

**Workaround:** If you need the reusable workflow's steps, call it explicitly from a non-reusable workflow and include that workflow.

### GitHub GraphQL API

The Octoshim proxy does not implement a GraphQL-to-REST translation layer. Actions that use `@octokit/graphql` or similar will receive HTTP 501.

### `pull_request_target`

This trigger gives workflows elevated permissions on fork PRs. GitLab's merge request CI does not have a direct analogue with the same security model. Not mapped.
