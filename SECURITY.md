# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x (latest) | ✅ |
| < 1.0 | ❌ |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitLab issues.**

Report vulnerabilities privately via GitLab's confidential issue feature:

1. Go to the [octolab issue tracker](https://gitlab.com/your-org/octolab/-/issues/new)
2. Check **"This issue is confidential"** before submitting
3. Use the title prefix `[SECURITY]`

You can also reach the maintainers directly via email (see the repository's contact information).

We aim to acknowledge reports within **48 hours** and provide a remediation timeline within **7 days**.

## Security Design

### Token Handling

The bridge is designed so that `GITLAB_PAT` — the only real credential in use — is never stored, logged, or exposed in artifacts:

1. **`::add-mask::` directives** are emitted before any other output, instructing `act`'s log processor to redact the PAT from all subsequent stdout.
2. **Inline `sed` filter** in `bridge.sh` replaces any literal occurrence of the PAT in the `act` output stream before writing to `bridge-logs/`.
3. **Octoshim's logging middleware** redacts the PAT value from all structured log entries.
4. **The generated child pipeline YAML** references `${GITLAB_PAT}` as a GitLab CI variable expansion, not the literal token value.
5. **The `GITHUB_TOKEN`** exposed to `act` is a random hex placeholder generated at runtime — it does not grant any real access. The Octoshim proxy intercepts all `Authorization` headers before they reach the GitLab API.

### Network Isolation

- Octoshim runs on `localhost:8080` inside the job container. It is not exposed to the GitLab runner host or other containers.
- All traffic to `GITHUB_API_URL` (which resolves to `localhost:8080`) stays within the container.
- Real outbound traffic to GitLab API uses `BRIDGE_GITLAB_URL` with the runner's standard network configuration.

### Image Pinning

- The `nektos/act` binary is downloaded in a dedicated build stage with a pinned version and SHA256 checksum verification (`sha256sum -c -`). Builds fail if the checksum does not match.
- We recommend pinning `BRIDGE_RUNNER_IMAGE` and `BRIDGE_DISPATCHER_IMAGE` to specific digest-based tags in production rather than using `:latest`.

### Least-Privilege Recommendations

- Grant `GITLAB_PAT` only the `api` scope. Do not use a token with `write_repository` or admin scopes.
- Mark `GITLAB_PAT` as **Protected** in addition to **Masked** to restrict it to protected branches/tags.
- Consider using project access tokens (scoped to the project) rather than personal access tokens where your GitLab instance supports them.

### Artifact Security

- `bridge-logs/` artifacts have `expire_in: 1 day` by default. Reduce this if log content is sensitive.
- The `generated-ci.yml` child pipeline artifact expires in 2 hours and contains only variable references (`${GITLAB_PAT}`), not literal values.
