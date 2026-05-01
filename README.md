# octolab — GitLab-to-GitHub Universal CI Bridge

[![pipeline status](https://gitlab.com/your-org/octolab/badges/main/pipeline.svg)](https://gitlab.com/your-org/octolab/-/pipelines)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)](CHANGELOG.md)

Run your existing GitHub Actions workflows on GitLab CI — **without changing a single line** of your `.github/workflows/` files.

```
Your project on GitLab
  └── .gitlab-ci.yml          ← adds one `include:` line
  └── .github/workflows/
        └── ci.yml            ← never touched, runs as-is
        └── release.yml       ← never touched, runs as-is
```

## How It Works

```
GitLab Push / MR event
        │
        ▼
┌─────────────────────┐
│  Stage A: Dispatcher │  Scans .github/workflows/, matches triggers,
│  (dispatcher.py)     │  writes generated-ci.yml child pipeline
└────────┬────────────┘
         │  dynamic child pipeline
         ▼
┌─────────────────────┐
│  Stage B: Executor   │  For each matched workflow:
│  (shim-runner image) │
│                      │  ┌──────────────────┐
│  bridge.sh           │  │  octoshim proxy  │ ← translates GitHub API
│    → maps env vars   │  │  :8080           │   calls to GitLab API
│    → starts proxy    │  └──────────────────┘
│    → calls `act`     │
│    → runs workflow   │
└─────────────────────┘
```

The shim image contains [`nektos/act`](https://github.com/nektos/act) as its execution engine. `act` parses your GitHub Actions YAML and resolves `uses:` Marketplace actions, while the **Octoshim** API proxy intercepts any GitHub API calls (check-runs, PR comments, etc.) and translates them to GitLab equivalents on the fly.

## Quickstart

**1. Set a masked CI/CD variable** in your GitLab project settings:

| Variable | Value | Options |
|----------|-------|---------|
| `GITLAB_PAT` | A GitLab Personal Access Token with `api` scope | Masked ✓ |

**2. Add one line to your `.gitlab-ci.yml`:**

```yaml
include:
  - project: 'your-org/octolab'
    ref: '1.0.0'
    file: 'universal-bridge.yml'
```

**3. Push.** The bridge automatically detects which of your GitHub Actions workflows match the current event and runs them.

That's it. Your `.github/workflows/` files stay untouched and keep working on GitHub too.

## Requirements

- GitLab CI with Docker executor
- GitLab runner with Docker-in-Docker (DinD) support
- `GITLAB_PAT` CI variable (masked, `api` scope)
- Your project already has `.github/workflows/*.yml` files

## Configuration

All configuration is done through CI/CD variables. None are required except `GITLAB_PAT`.

| Variable | Default | Description |
|----------|---------|-------------|
| `GITLAB_PAT` | — | **Required.** GitLab PAT with `api` scope. Must be masked. |
| `BRIDGE_RUNNER_IMAGE` | `octolab/shim-runner:latest` | Docker image for the shim executor |
| `BRIDGE_DISPATCHER_IMAGE` | `octolab/bridge-dispatcher:latest` | Docker image for the dispatcher |
| `BRIDGE_WORKFLOW_DIR` | `.github/workflows` | Directory to scan for workflow files |
| `BRIDGE_OUTPUT_FILE` | `generated-ci.yml` | Path for the generated child pipeline |
| `BRIDGE_LOG_LEVEL` | `info` | Proxy log verbosity (`debug`/`info`/`warn`) |

Override any variable in your own `.gitlab-ci.yml`:

```yaml
include:
  - project: 'your-org/octolab'
    ref: '1.0.0'
    file: 'universal-bridge.yml'

variables:
  BRIDGE_LOG_LEVEL: debug
  BRIDGE_RUNNER_IMAGE: registry.gitlab.com/your-org/octolab/shim-runner:1.0.0
```

## Event Mapping

| GitLab `CI_PIPELINE_SOURCE` | GitHub `GITHUB_EVENT_NAME` |
|-----------------------------|---------------------------|
| `push` | `push` |
| `merge_request_event` | `pull_request` |
| `external_pull_request_event` | `pull_request` |
| `web` / `api` / `chat` | `workflow_dispatch` |
| `trigger` | `repository_dispatch` |
| `pipeline` / `parent_pipeline` | `workflow_run` |
| `schedule` | `schedule` |

## Supported Features

| Feature | Status |
|---------|--------|
| `push` triggers (branches, tags, globs) | ✅ Supported |
| `pull_request` triggers (branches, globs) | ✅ Supported |
| `workflow_dispatch` | ✅ Supported |
| `schedule` | ✅ Supported |
| `uses:` Marketplace actions | ✅ via `act` |
| Docker actions (`runs: docker`) | ✅ via DinD |
| `$GITHUB_OUTPUT` / `$GITHUB_ENV` | ✅ Supported |
| Job artifacts | ✅ via `act` artifact server |
| PR comments (`github-script`) | ✅ via Octoshim proxy |
| Check-run status updates | ✅ via Octoshim proxy |
| `paths:` / `paths-ignore:` filters | ⚠️ Planned (v2) — currently match-all |
| `workflow_call` (reusable workflows) | ❌ Not a CI event trigger |
| GitHub Environments / Deployments | ❌ No GitLab equivalent |
| GitHub GraphQL API | ❌ Not supported |

See [docs/SUPPORTED-FEATURES.md](docs/SUPPORTED-FEATURES.md) for the full compatibility matrix.

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for a detailed breakdown of how the Dispatcher, Executor, and Octoshim proxy interact.

## Security

- `GITLAB_PAT` is never written to logs or artifacts. The bridge emits `::add-mask::` directives and applies a `sed` filter to the `act` output stream before writing log files.
- The `GITHUB_TOKEN` passed to `act` is a random placeholder — the Octoshim proxy intercepts all `Authorization` headers and replaces them with the real GitLab credentials before forwarding.
- See [SECURITY.md](SECURITY.md) to report a vulnerability.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and the PR process.

## License

[MIT](LICENSE) © octolab contributors
