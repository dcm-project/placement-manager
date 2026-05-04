# Contributing to DCM Placement Manager

## Prerequisites

See [README.md](README.md#prerequisites) for Go, PostgreSQL, and container
runtime requirements.

## Secret Scanning

This repository uses [gitleaks](https://github.com/gitleaks/gitleaks) to
prevent credentials from being committed. CI will block any PR that introduces
secrets.

### Local pre-commit hook (recommended)

Install the [pre-commit](https://pre-commit.com/) framework and enable the
hook so secrets are caught before they reach CI:

```bash
pip install pre-commit   # or: brew install pre-commit
pre-commit install
```

After this, every `git commit` will automatically run gitleaks against your
staged changes. If a false positive is flagged, see the allowlist in
`.gitleaks.toml`.

### Updating the gitleaks version

The gitleaks version is pinned in two places:

| File | Field |
|------|-------|
| `.github/workflows/secret-scan.yaml` | `env.GITLEAKS_VERSION` |
| `.pre-commit-config.yaml` | `rev` |

When updating, change both to the same version tag.

## Running Tests

```bash
make test                   # Unit tests
make subsystem-test-full    # Integration tests (requires Podman/Docker)
```

## Code Generation

After modifying the OpenAPI spec in `api/v1alpha1/openapi.yaml`:

```bash
make generate-api
make check-generate-api     # Verify generated files are in sync
```

## Pull Request Checklist

- [ ] `make test` passes
- [ ] `make vet` and `make fmt` produce no changes
- [ ] Generated files are in sync (`make check-generate-api`)
- [ ] No secrets or credentials in committed code
