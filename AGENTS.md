# AGENTS.md

## Overview
`dev-vault` is a Go CLI that pulls/pushes Scaleway Secret Manager `*-dev` secrets to disk for local development workflows.

Configuration is per-project via a committed `.scw.json` manifest that maps secret names to local file paths relative to the project root.

This file documents project conventions and hard constraints for contributors and automation/agents.

## Hard Safety Rules
- Never print secret payloads (to stdout/stderr, logs, or error messages).
- Refuse to operate on any secret name that does not end with `-dev` (this is an invariant, not a convention).
- Do not commit credentials, tokens, or real secret identifiers into the repository (including README examples).
- If a secret/token is ever pasted into chat/logs, treat it as compromised and rotate/revoke immediately.

## Common Commands
- Install: `go install github.com/bsmartlabs/dev-vault/cmd/dev-vault@latest`
- Test (100% coverage required): `go test ./... -coverprofile=coverage.out`
- Coverage summary: `go tool cover -func=coverage.out | tail -n 1`
- Run help: `go run ./cmd/dev-vault -h`

## `.scw.json` Notes (v1)
- File name is fixed: `.scw.json`.
- Discovery: the CLI searches upward from the current working directory until it finds `.scw.json` (or pass `--config <path>`).
- The directory containing `.scw.json` is the "project root"; all `mapping.*.file` paths are relative to that root.
- `mapping` keys:
  - Are Scaleway secret names.
  - Must end in `-dev` (hard enforced).
- `mapping[*].format`:
  - `raw`: secret bytes are written as-is.
  - `dotenv`: secret payload is expected to be a JSON object; it is rendered deterministically as a `.env` style file.
- `mapping[*].mode`:
  - `both` (default): eligible for both `pull --all` and `push --all`.
  - `pull`: only eligible for `pull --all`.
  - `push`: only eligible for `push --all`.
  - Legacy: `sync` is accepted as an alias for `both`.

## CI Local Runs (GitHub Actions via `act`)
- Test job: `act -W .github/workflows/ci.yml -j test`
- Build job: `act -W .github/workflows/ci.yml -j build`
- Apple Silicon: add `--container-architecture linux/arm64`

## CI (GitHub Actions)
- `ci` runs on:
  - `pull_request` (all PRs, including Renovate PRs)
  - `push` to `main` only
- The CI pipeline gates with:
  - gitleaks scan (downloads the latest gitleaks release at runtime)
  - `go test` with 100.0% statement coverage enforced
  - multi-arch build smoke test (`linux/darwin/windows`, `amd64/arm64` where applicable)

## Homebrew Tap Update
Publishing is done on Git tags (`v*`) via GoReleaser, and the release workflow attempts to update the Homebrew tap formula.

Tap update requires `HOMEBREW_TAP_GITHUB_TOKEN` to have write access to `bsmartlabs/homebrew-dev-tools`.

Manual tap update can be done with `scripts/update-homebrew-formula.sh` after a GitHub release exists with `checksums.txt`:

```bash
scripts/update-homebrew-formula.sh \
  --repo bsmartlabs/dev-vault \
  --tag v1.2.3 \
  --tap bsmartlabs/homebrew-dev-tools \
  --token "$HOMEBREW_TAP_GITHUB_TOKEN"
```

## Releases
- Release workflow trigger: pushing a tag matching `v*`.
- Weekly auto-tagging exists as a workflow, but scheduled runs are disabled by default:
  - Set repo variable `DEV_VAULT_ENABLE_SCHEDULED_RELEASES=true` to enable scheduled tagging.
  - `workflow_dispatch` can also be used to create a tag explicitly.

## Renovate
- Renovate is enabled (`renovate.json`) and configured to automerge safe updates when CI is green.
- Avoid introducing workflow triggers that run on every branch push; keep `push` workflows limited to `main` to avoid Renovate noise/cost.
