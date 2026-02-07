# dev-vault

`dev-vault` is a Go CLI for pulling and pushing Scaleway Secret Manager secrets to disk for local development workflows.

It is configured per-project via a committed `.scw.json` manifest that maps secret names (must end with `-dev`) to files relative to the project root.

## Install

### Homebrew (macOS)

```bash
brew tap bsmartlabs/dev-tools
brew install dev-vault
```

### From source (Go)

```bash
go install github.com/bsmartlabs/dev-vault/cmd/dev-vault@latest
```

## Auth

Authentication is done via the Scaleway Go SDK (no dependency on the `scw` CLI binary). Credentials can come from:

- Environment variables (e.g. `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`)
- `~/.config/scw/config.yaml` profiles (set `profile` in `.scw.json` or use `--profile`)

Note: `.scw.json` is JSON and is the only required config file for `dev-vault`. The YAML file above is the standard Scaleway profile config used by Scaleway tooling/SDKs.

## `.scw.json` (v1)

`dev-vault` searches upward from the current directory for `.scw.json` (or you can pass `--config <path>`).

Example:

```json
{
  "organization_id": "01234567-89ab-cdef-0123-456789abcdef",
  "project_id": "89abcdef-0123-4567-89ab-cdef01234567",
  "region": "fr-par",
  "profile": "default",
  "mapping": {
    "bweb-env-bsmart-dev": {
      "file": ".env.bsmart.rework",
      "format": "dotenv",
      "path": "/",
      "type": "key_value"
    },
    "some-cert-dev": {
      "file": "certs/dev.pem",
      "format": "raw",
      "mode": "pull"
    }
  }
}
```

Notes:

- `mapping` keys are Scaleway secret names and must end with `-dev` (hard enforced).
- `file` paths are relative to the directory containing `.scw.json` and cannot escape the project root.
- Secret payloads are never printed.

## Safety Constraints

- Refuses to operate on any secret that does not end with `-dev`.
- Never prints secret payloads to stdout/stderr.

## Commands

```bash
dev-vault version
dev-vault list [--name-contains <s> ...] [--name-regex <re>] [--path <p>] [--type <t>] [--json]
dev-vault pull (--all | <secret-dev> ...) [--overwrite]
dev-vault push (--all | <secret-dev> ...) [--yes] [--disable-previous] [--description <s>] [--create-missing]
```

## Development

Unit tests are fully mocked (no Scaleway network calls).

Tests require 100% statement coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -n 1
```

CI runs on every push and PR and builds multi-arch binaries.

To test GitHub Actions locally with `act`:

```bash
act -W .github/workflows/ci.yml -j test
act -W .github/workflows/ci.yml -j build
```

On Apple Silicon, you may need:

```bash
act -W .github/workflows/ci.yml -j test --container-architecture linux/arm64
act -W .github/workflows/ci.yml -j build --container-architecture linux/arm64
```
