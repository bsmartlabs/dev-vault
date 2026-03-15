# dev-vault

`dev-vault` is a Go CLI that syncs Scaleway Secret Manager `*-dev` secrets with files in a local project.

It is designed for development workflows where teams need deterministic pull/push operations driven by a committed project manifest (`.scw.json`), without ever printing secret payloads.

## What It Does

- Pulls secrets from Scaleway into local files (`pull`).
- Pushes local files as new secret versions in Scaleway (`push`).
- Lists `*-dev` secret metadata in a project (`list`).
- Enforces strict safety invariants:
  - secret names must end with `-dev`
  - file paths must stay inside project root
  - payload values are never printed

## Why It Exists

`dev-vault` keeps local dev secret handling explicit and reviewable:

- Secret/file mappings are versioned in code (`.scw.json`).
- Bulk operations are controlled with `mapping.mode`.
- Dotenv conversion is deterministic (stable output ordering).
- CI validates behavior with 100% statement coverage.

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

## Authentication

`dev-vault` uses the Scaleway Go SDK directly (no `scw` CLI dependency).

Credentials can come from:

- Environment variables (for example `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`)
- Scaleway profiles in `~/.config/scw/config.yaml`

Profile resolution precedence:

1. `--profile <name>` CLI flag
2. `.scw.json` `profile`
3. SDK environment/default behavior

## How It Works

1. Resolve config:
   - use `--config <path>` if provided
   - otherwise search upward from cwd for `.scw.json`
2. Validate policy:
   - `pull`/`push`: require a valid `mapping`
   - `list`: only requires project fields (`organization_id`, `project_id`, `region`)
3. Select targets:
   - explicit names, or `--all` filtered by `mapping.mode`
4. Sync:
   - `pull`: read latest enabled version and write atomically to disk
   - `push`: read local file and create a new secret version
5. Report:
   - print only metadata/status and exit with contract-based status code

## `.scw.json` (v1)

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
      "mode": "pull",
      "type": "key_value"
    },
    "some-cert-dev": {
      "file": "certs/dev.pem",
      "format": "raw",
      "mode": "push"
    }
  }
}
```

Rules:

- Config filename is fixed: `.scw.json`.
- `mapping` keys are Scaleway secret names and must end with `-dev`.
- `mapping[*].file` must be relative and cannot escape project root.
- `mapping[*].mode` is required: `pull` or `push`.
- `mapping[*].mode` is required: `pull`, `push`, or `skip`.
- `mapping[*].format` defaults to `raw` (`raw` or `dotenv`).
- `mapping[*].path` defaults to `/`.
- `mapping[*].type` is optional, but required when using `push --create-missing`.
- Unknown JSON fields and trailing JSON data are rejected.

## Command Reference

```bash
dev-vault version
dev-vault list [--name-contains <s> ...] [--name-regex <re>] [--path <p>] [--type <t>] [--json]
dev-vault pull (--all | <secret-dev> ...)
dev-vault push (--all | <secret-dev> ...) [--yes] [--disable-previous] [--description <s>] [--create-missing]
dev-vault help [command]
```

Notes:

- Global flags can be passed before or after the command:
  - `dev-vault --config .scw.json pull x-dev`
  - `dev-vault pull --config .scw.json x-dev`
- `list` always filters by secret name suffix `-dev`.
- `pull --all` includes only entries where `mapping.mode=pull`.
- `push --all` includes only entries where `mapping.mode=push`.
- `mapping.mode=skip` is ignored by both `pull --all` and `push --all`.
- `push` requires `--yes` when pushing more than one secret.
- `push --create-missing` creates missing secrets using `mapping.type` and `mapping.path`.
- `pull` overwrites existing targets and creates missing parent directories.

## Behavior Guarantees

- Secret payloads are never printed to stdout/stderr.
- `pull` writes atomically and applies mode `0600` on Unix.
- `pull` overwrites an existing target path by default.
- `pull` reads revision selector `latest_enabled`.
- Dotenv handling:
  - pull: JSON object payload -> deterministic `.env`
  - push: `.env` -> JSON object payload

## Exit Codes

- `0`: success
- `1`: runtime/output/internal error
- `2`: usage/argument/validation error

Batch commands (`pull`/`push`) can be partially successful; if at least one target fails, command exits non-zero and reports failures by secret name.

## Development

Unit tests are mocked by default (no live Scaleway calls).

Provider compatibility gate:

- Contract policy and test layout: [docs/contracts/provider-compatibility.md](docs/contracts/provider-compatibility.md)
- Optional live read-only integration gate:

```bash
DEV_VAULT_TEST_PROJECT_ID=<project-id> \
DEV_VAULT_TEST_ORGANIZATION_ID=<org-id> \
DEV_VAULT_TEST_REGION=fr-par \
scripts/test-provider-contract.sh
```

Default quality-gate order:

1. `go test ./... -coverprofile=coverage.out`
2. `go tool cover -func=coverage.out | tail -n 1` (must be `100.0%`)
3. `make test-contracts` (or `scripts/test-provider-contract.sh`)

To skip provider live contracts explicitly in local/contributor flows:

```bash
ALLOW_CONTRACT_SKIP=1 make test-contracts
```

## CI

CI runs on:

- `pull_request`
- `push` to `main`

Pipeline gates:

- gitleaks scan
- `go test` with 100.0% statement coverage enforcement
- provider contract checks
- multi-arch build smoke test (`linux/darwin/windows`, `amd64/arm64` where applicable)

Run GitHub Actions locally with `act`:

```bash
act -W .github/workflows/ci.yml -j test
act -W .github/workflows/ci.yml -j build
```

On Apple Silicon:

```bash
act -W .github/workflows/ci.yml -j test --container-architecture linux/arm64
act -W .github/workflows/ci.yml -j build --container-architecture linux/arm64
```
