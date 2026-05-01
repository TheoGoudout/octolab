# Octoshim API Proxy

Octoshim is a lightweight HTTP proxy that intercepts GitHub API calls made by workflow steps and translates them into equivalent GitLab API calls.

## How It Works

When `bridge.sh` starts, it sets `GITHUB_API_URL=http://localhost:8080` before invoking `act`. Any workflow step or Marketplace action that calls the GitHub API (e.g. `actions/github-script`, `peter-evans/create-pull-request`) will hit Octoshim instead of `api.github.com`.

Octoshim then:
1. Strips the `Authorization: token <placeholder>` header
2. Injects `PRIVATE-TOKEN: <GITLAB_PAT>`
3. Translates the request path and body to the GitLab API equivalent
4. Forwards the request to `BRIDGE_GITLAB_URL/api/v4`
5. Remaps the GitLab response to look like a GitHub API response
6. Returns the remapped response to `act`

## Endpoint Reference

### Health Check

```
GET /health  →  200 OK
```

Used by `bridge.sh` to poll for proxy readiness before starting `act`.

### Check Runs

```
POST  /repos/:owner/:repo/check-runs
PATCH /repos/:owner/:repo/check-runs/:check_run_id
```

**Translates to:** `POST /api/v4/projects/:id/statuses/:sha`

**Status mapping:**

| GitHub `status` | GitHub `conclusion` | GitLab `state` |
|----------------|---------------------|----------------|
| `queued` | — | `pending` |
| `in_progress` | — | `running` |
| `completed` | `success` / `skipped` / `neutral` | `success` |
| `completed` | `failure` / `timed_out` / `action_required` | `failed` |
| `completed` | `cancelled` | `canceled` |

**Request body (GitHub):**
```json
{
  "name": "My Check",
  "head_sha": "abc123",
  "status": "completed",
  "conclusion": "success",
  "details_url": "https://example.com/logs",
  "output": {
    "title": "Tests passed",
    "summary": "All 42 tests passed."
  }
}
```

**Response body (GitHub-shaped):**
```json
{
  "id": 12345,
  "name": "My Check",
  "status": "completed",
  "html_url": ""
}
```

### Issue Comments (→ MR Notes)

```
POST /repos/:owner/:repo/issues/:issue_number/comments
GET  /repos/:owner/:repo/issues/:issue_number/comments
```

`POST` **translates to:** `POST /api/v4/projects/:id/merge_requests/:iid/notes`

The `:issue_number` in the GitHub path is used as the MR `iid` on GitLab.

**Request body (GitHub):**
```json
{ "body": "Hello from a GitHub Action!" }
```

**Response body (GitHub-shaped):**
```json
{
  "id": 67890,
  "body": "Hello from a GitHub Action!",
  "user": { "login": "gitlab-username" }
}
```

`GET` returns an empty array `[]`. Reading comments is informational; most actions only need to create them.

### Pull Requests (→ Merge Requests)

```
GET /repos/:owner/:repo/pulls/:pull_number
GET /repos/:owner/:repo/pulls
```

**Translates to:**
- `GET /api/v4/projects/:id/merge_requests/:iid`
- `GET /api/v4/projects/:id/merge_requests[?state=opened]`

**Key field remapping:**

| GitLab field | GitHub field | Notes |
|-------------|-------------|-------|
| `iid` | `number` | GitLab's per-project MR ID |
| `id` | `id` | GitLab's global ID |
| `description` | `body` | |
| `source_branch` | `head.ref` | |
| `target_branch` | `base.ref` | |
| `sha` | `head.sha` | Latest commit SHA |
| `opened` | `open` | State translation |
| `merged` / `closed` | `closed` | State translation |
| `merge_status == "can_be_merged"` | `mergeable: true` | |
| `web_url` | `html_url` | |

**Query parameter translation:**

`?state=open` → `?state=opened` for the GitLab API.

### Issues

```
POST /repos/:owner/:repo/issues
```

**Translates to:** `POST /api/v4/projects/:id/issues`

**Request body (GitHub):**
```json
{ "title": "My Issue", "body": "Description here" }
```

GitLab's `description` field is used for the issue body.

### Commits

```
GET /repos/:owner/:repo/commits/:sha
```

**Translates to:** `GET /api/v4/projects/:id/repository/commits/:sha`

Returns a GitHub-shaped response with `sha`, `html_url`, and `commit.author`.

### Labels

```
GET /repos/:owner/:repo/labels
```

**Translates to:** `GET /api/v4/projects/:id/labels`

Returns an array of `{ "id": ..., "name": "..." }` objects.

## Unsupported Endpoints

Any request that does not match a known route returns:

```
HTTP 501 Not Implemented

{
  "error": "NOT SUPPORTED",
  "endpoint": "POST /repos/owner/repo/environments",
  "message": "This GitHub API endpoint has no equivalent in the GitLab bridge. The workflow step will fail but other jobs will continue."
}
```

A structured warning is also written to the Octoshim log:
```
level=WARN msg="unsupported GitHub API endpoint" method=POST path=/repos/owner/repo/environments
```

Known unsupported categories:
- `/repos/.../environments` — GitHub Environments
- `/repos/.../deployments` — GitHub Deployments
- `/repos/.../releases` — GitHub Releases
- `/graphql` — GitHub GraphQL API
- All other unrecognised paths

## Authentication

Octoshim implements an auth-swap middleware:

**Incoming (from `act`):**
```
Authorization: token bridge-token-<random>
```

**Outgoing (to GitLab):**
```
PRIVATE-TOKEN: <GITLAB_PAT value>
```

The `Authorization` header is stripped before forwarding. The `PRIVATE-TOKEN` value is never logged; the logging middleware redacts it with `[MASKED]`.

## Running Octoshim Locally

For testing or debugging:

```bash
export BRIDGE_GITLAB_URL=https://gitlab.com
export BRIDGE_GITLAB_PAT=glpat-xxxx
export BRIDGE_GITLAB_PROJECT_ID=12345678
export BRIDGE_GITLAB_PROJECT_PATH=your-group/your-project
export BRIDGE_LOG_LEVEL=debug

./octoshim/octoshim
# Listens on :8080

# Test health
curl http://localhost:8080/health

# Test PR lookup
curl -H "Authorization: token fake" \
     http://localhost:8080/repos/owner/repo/pulls/1
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BRIDGE_GITLAB_URL` | (required) | GitLab instance URL, e.g. `https://gitlab.com` |
| `BRIDGE_GITLAB_PAT` | (required) | PAT with `api` scope |
| `BRIDGE_GITLAB_PROJECT_ID` | (required) | Numeric project ID or URL-encoded path |
| `BRIDGE_GITLAB_PROJECT_PATH` | (required) | `group/project` path |
| `BRIDGE_LOG_LEVEL` | `info` | `debug` / `info` / `warn` |
| `OCTOSHIM_ADDR` | `:8080` | Listen address |
