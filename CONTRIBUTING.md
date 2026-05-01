# Contributing

Thank you for considering a contribution to octolab. This guide covers how to set up a development environment, run tests, and submit changes.

## Table of Contents

- [Development Setup](#development-setup)
- [Project Layout](#project-layout)
- [Running Tests](#running-tests)
- [Coding Conventions](#coding-conventions)
- [Submitting a Merge Request](#submitting-a-merge-request)
- [Release Process](#release-process)

## Development Setup

### Prerequisites

- Docker (for building the shim image)
- Go 1.22+
- Python 3.11+
- `make`

### Clone and install

```bash
git clone https://gitlab.com/your-org/octolab.git
cd octolab

# Install Python test dependencies
pip install pyyaml==6.0.1 pytest

# Build the Go proxy
cd octoshim && go build ./...
```

### Quick smoke test

```bash
make test
```

## Project Layout

```
octolab/
├── dispatcher/          Python dispatcher (Stage A)
│   ├── dispatcher.py    Core logic
│   ├── requirements.txt Python deps
│   └── tests/           pytest suite + YAML fixtures
├── entrypoint/          Shell/Python scripts baked into the shim image
│   ├── bridge.sh        Main entrypoint
│   ├── mask-tokens.sh   Token masking + env export
│   └── generate-event.py  GitHub event JSON generator
├── octoshim/            Go API proxy (Stage B)
│   ├── main.go          Server entry point
│   ├── router.go        URL dispatch
│   ├── handlers/        One file per endpoint group
│   ├── gitlab/          GitLab API client + types
│   ├── transform/       Request/response mapping
│   └── middleware/      Auth swap + structured logging
├── docs/                Reference documentation
├── examples/            Example project integrations
├── Dockerfile           Multi-stage shim runner image
├── universal-bridge.yml Master GitLab CI include file
├── .gitlab-ci.yml       Bridge's own CI pipeline
└── Makefile             Developer commands
```

## Running Tests

### All tests

```bash
make test
```

### Dispatcher (Python)

```bash
make test-dispatcher
# or directly:
pytest dispatcher/tests/ -v
```

### Octoshim (Go)

```bash
make test-octoshim
# or directly:
cd octoshim && go test ./... -v
```

### Shell linting

```bash
make lint-shell
# Requires shellcheck to be installed
```

### Building the Docker image

```bash
make docker-build
```

## Coding Conventions

### Python (`dispatcher/`)

- Follow PEP 8. The project uses 4-space indentation.
- All public functions must have a short docstring.
- New trigger types must be covered by a test in `dispatcher/tests/test_dispatcher.py`.
- New fixtures go in `dispatcher/tests/fixtures/`.

### Go (`octoshim/`)

- `gofmt` formatting is required. Run `make fmt` before committing.
- `go vet ./...` must pass with zero warnings.
- New API endpoints require:
  - A handler file in `octoshim/handlers/`
  - A route entry in `octoshim/router.go` (longest-first order)
  - A corresponding entry in [docs/API-PROXY.md](docs/API-PROXY.md)
  - An entry in the supported features table in [docs/SUPPORTED-FEATURES.md](docs/SUPPORTED-FEATURES.md)
- Do not add external dependencies. The proxy must remain stdlib-only.

### Shell (`entrypoint/`)

- `shellcheck` must pass for `bridge.sh` and `mask-tokens.sh`.
- Use `set -euo pipefail` at the top of every script.
- Quote all variable expansions.

### Documentation

- Update [docs/SUPPORTED-FEATURES.md](docs/SUPPORTED-FEATURES.md) whenever you add or change a supported feature.
- Update [CHANGELOG.md](CHANGELOG.md) under the `[Unreleased]` heading.

## Submitting a Merge Request

1. Fork the repository and create a branch: `git checkout -b feat/my-feature`
2. Make your changes, following the conventions above.
3. Run `make test` and ensure all tests pass.
4. Update `CHANGELOG.md` under `[Unreleased]`.
5. Open a merge request against the `main` branch.
6. Fill in the MR description template.

### What makes a good MR

- Focused: one logical change per MR
- Tested: new behaviour covered by unit tests
- Documented: `SUPPORTED-FEATURES.md` and `API-PROXY.md` updated if applicable
- Clean: `make fmt lint` passes

## Release Process

Releases are tagged versions. The bridge images are published to the container registry automatically when a `vX.Y.Z` tag is pushed.

1. Update `CHANGELOG.md`: move `[Unreleased]` items under the new version heading
2. Commit: `git commit -m "chore: release v1.1.0"`
3. Tag: `git tag v1.1.0`
4. Push: `git push origin main v1.1.0`
5. The `.gitlab-ci.yml` `publish:versioned-tag` job will build and push the tagged Docker images
