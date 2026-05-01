# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] — 2026-04-30

### Added

- **Dispatcher** (`dispatcher/dispatcher.py`): Python script that scans `.github/workflows/`, maps GitLab CI events to GitHub trigger types, evaluates `branches`, `tags`, `branches-ignore`, and `tags-ignore` glob filters, and writes a dynamic child pipeline YAML. Handles the PyYAML `on:`→`True` boolean key quirk.
- **Octoshim API proxy** (`octoshim/`): Go HTTP proxy (stdlib-only) that translates GitHub API calls to GitLab API calls at runtime:
  - `POST /repos/.../check-runs` → GitLab commit statuses
  - `POST /repos/.../issues/:n/comments` → GitLab MR notes
  - `GET/POST /repos/.../pulls[/:n]` → GitLab merge requests
  - Issues, commits, and labels endpoints
  - HTTP 501 + structured log for unsupported endpoints (environments, deployments, GraphQL)
- **Entrypoint scripts** (`entrypoint/`):
  - `bridge.sh`: maps all GitLab CI variables to GitHub Actions equivalents, starts octoshim, generates event JSON, invokes `nektos/act`, and masks `GITLAB_PAT` from log output
  - `mask-tokens.sh`: validates `GITLAB_PAT`, emits `::add-mask::` directives
  - `generate-event.py`: builds GitHub-compatible event JSON payloads for push, pull_request, workflow_dispatch, schedule, repository_dispatch, and workflow_run events
- **Shim runner image** (`Dockerfile`): multi-stage build producing an Ubuntu 24.04 image with `act`, `octoshim`, Python 3, Node.js, Docker client, jq, and git
- **Universal bridge include file** (`universal-bridge.yml`): single `include:` target for user projects
- **37 unit tests** for the dispatcher covering all trigger types and edge cases
- Full documentation: README, ARCHITECTURE, CONFIGURATION, SUPPORTED-FEATURES, API-PROXY, CONTRIBUTING

### Supported trigger types

`push`, `pull_request`, `workflow_dispatch`, `schedule`, `repository_dispatch`, `workflow_run`

### Known limitations in this release

- `paths:` and `paths-ignore:` trigger filters are not evaluated (treated as match-all; planned for v2)
- `workflow_dispatch.inputs` are not forwarded from the GitLab pipeline trigger
- GitHub Environments, Deployments, and GraphQL API are not supported
- Cross-job artifact caching is local to the job container only

[Unreleased]: https://gitlab.com/your-org/octolab/-/compare/v1.0.0...HEAD
[1.0.0]: https://gitlab.com/your-org/octolab/-/releases/v1.0.0
