# Minimal Bridge Example

This example shows the smallest possible integration with the octolab bridge.

## Files

```
.gitlab-ci.yml                     ← one include: line, nothing else
.github/workflows/ci.yml           ← your existing GitHub Actions workflow, untouched
```

## What happens on a push to `main`

1. GitLab runs `bridge-dispatch` in the `.pre` stage, which reads `ci.yml` and sees it matches `push` events targeting `main`.
2. A child pipeline is generated with a single job: `bridge_ci`.
3. `bridge_ci` starts the shim runner image, maps all GitLab CI variables to GitHub Actions variables, and runs `act` against `.github/workflows/ci.yml`.
4. `actions/checkout@v4` clones the repository into `$GITHUB_WORKSPACE`.
5. `actions/setup-node@v4` installs Node 20.
6. `npm ci && npm test` runs.
7. On a pull request event, `actions/github-script@v7` posts a comment via the Octoshim proxy → GitLab MR note.

## What happens on a push to a feature branch

The dispatcher sees that `push.branches: [main, develop]` does not match the feature branch, so **no bridge jobs are generated**. The pipeline completes immediately with a no-op job.

## Prerequisites

- `GITLAB_PAT` set as a masked CI/CD variable
- GitLab runner with Docker executor in privileged mode
