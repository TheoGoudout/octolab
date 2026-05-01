# Configuration Reference

All bridge configuration is done through GitLab CI/CD variables. Set them in your project's **Settings → CI/CD → Variables**.

## Required Variables

### `GITLAB_PAT`

A GitLab Personal Access Token used by Octoshim to call the GitLab API on behalf of the running workflow.

| Setting | Value |
|---------|-------|
| Type | Variable |
| Flags | **Masked** (required), Protected (recommended) |
| Scope | All environments, or restrict to protected branches |

**Required scopes:** `api`

**How to create:**
1. Go to your GitLab profile → **Access Tokens**
2. Name: `octolab-bridge`
3. Scopes: check `api`
4. Copy the token value
5. Add it as a masked CI/CD variable named `GITLAB_PAT` in your project settings

## Optional Variables

All optional variables have sensible defaults and do not need to be set for basic usage.

### `BRIDGE_RUNNER_IMAGE`

The Docker image used to run each GitHub Actions workflow (the Stage B executor).

| Default | `octolab/shim-runner:latest` |
|---------|------------------------------|
| Example | `registry.gitlab.com/your-org/octolab/shim-runner:1.0.0` |

Pin this to a specific version tag in production to ensure reproducible builds.

### `BRIDGE_DISPATCHER_IMAGE`

The Docker image that runs `dispatcher.py` in the `.pre` stage.

| Default | `octolab/bridge-dispatcher:latest` |
|---------|-------------------------------------|
| Example | `registry.gitlab.com/your-org/octolab/bridge-dispatcher:1.0.0` |

### `BRIDGE_WORKFLOW_DIR`

Directory (relative to the project root) to scan for GitHub Actions workflow files.

| Default | `.github/workflows` |
|---------|---------------------|
| Example | `actions/workflows` |

### `BRIDGE_OUTPUT_FILE`

Path for the generated dynamic child pipeline YAML.

| Default | `generated-ci.yml` |
|---------|---------------------|

There is rarely a reason to change this.

### `BRIDGE_LOG_LEVEL`

Controls the verbosity of the Octoshim proxy's structured log output.

| Default | `info` |
|---------|--------|
| Values | `debug`, `info`, `warn` |

Use `debug` when troubleshooting API translation issues. At `debug` level, every HTTP request and response status is logged (but secret values are always redacted).

## Example Configuration

Minimal `.gitlab-ci.yml` with all options shown:

```yaml
include:
  - project: 'your-org/octolab'
    ref: '1.0.0'
    file: 'universal-bridge.yml'

# Optional overrides
variables:
  BRIDGE_RUNNER_IMAGE: registry.gitlab.com/your-org/octolab/shim-runner:1.0.0
  BRIDGE_DISPATCHER_IMAGE: registry.gitlab.com/your-org/octolab/bridge-dispatcher:1.0.0
  BRIDGE_LOG_LEVEL: info
```

Variables set in your project override the defaults in `universal-bridge.yml`.

## GitLab Runner Requirements

The runner executing the bridge jobs must:

1. **Use the Docker executor** — shell and Kubernetes executors are not supported.
2. **Have Docker-in-Docker (DinD) access** — the generated child pipeline jobs declare `services: docker:dind`. Your runner must allow privileged mode or have a pre-configured DinD socket.
3. **Have internet access** — `act` downloads Marketplace action repositories and runner container images on first use. Subsequent runs benefit from Docker layer caching.

### Privileged mode

To enable DinD, your GitLab runner's `config.toml` must include:

```toml
[[runners]]
  [runners.docker]
    privileged = true
```

### Caching act images

`act` uses `catthehacker/ubuntu:act-*` as the default platform image. These images are large (~1 GB). Pre-pull them in your runner's base image or configure a pull-through registry cache to avoid repeated downloads.

## Environment Variables Available Inside Workflows

The following variables are set inside the shim container and are available to all workflow steps:

```
GITHUB_WORKSPACE        /path/to/project
GITHUB_REPOSITORY       group/project
GITHUB_REPOSITORY_NAME  project
GITHUB_REPOSITORY_OWNER group
GITHUB_SHA              <commit sha>
GITHUB_REF              refs/heads/main  (or refs/tags/v1.0.0)
GITHUB_REF_NAME         main
GITHUB_HEAD_REF         feature-branch   (MR events only)
GITHUB_BASE_REF         main             (MR events only)
GITHUB_EVENT_NAME       push
GITHUB_EVENT_PATH       /tmp/github_event.json
GITHUB_RUN_ID           <CI_PIPELINE_ID>
GITHUB_RUN_NUMBER       <CI_PIPELINE_IID>
GITHUB_JOB              <CI_JOB_NAME>
GITHUB_ACTIONS          true
GITHUB_API_URL          http://localhost:8080
GITHUB_SERVER_URL       http://localhost:8080
GITHUB_TOKEN            <random placeholder — proxy intercepts auth>
GITHUB_OUTPUT           /github/output/output.txt
GITHUB_ENV              /github/output/env.txt
GITHUB_PATH             /github/output/path.txt
GITHUB_STEP_SUMMARY     /github/output/step_summary.md
RUNNER_NAME             gitlab-bridge-<runner-id>
RUNNER_WORKSPACE        /path/to/project
RUNNER_TEMP             /tmp/runner
ACTIONS_RUNTIME_TOKEN   <CI_JOB_TOKEN>
```
