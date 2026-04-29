# PROJECT KNOWLEDGE BASE

**Generated:** 2026-04-29
**Commit:** `5fe4e23`
**Branch:** `main`

## OVERVIEW
`dev-vault` is a Go 1.26 CLI that syncs Scaleway Secret Manager `*-dev` secrets with local files through a committed `.scw.json` manifest. It is safety-first: metadata may be printed, secret payload bytes never may be.

## STRUCTURE
```text
dev-vault/
├── cmd/dev-vault/                 # thin main: ldflags + real Scaleway opener + cli.Run
├── internal/cli/                  # Cobra wiring, exit codes, reporting, command runtime
├── internal/secretsync/           # provider-independent list/pull/push behavior
├── internal/config/               # .scw.json discovery, strict decode, normalization
├── internal/secretprovider/       # provider interface, contracts, Scaleway adapter
├── internal/testdouble/secretapi/ # shared stateful SecretAPI fake
├── docs/contracts/                # provider compatibility policy
└── scripts/                       # provider contract and package repo update scripts
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| CLI entry / exit behavior | `cmd/dev-vault/main.go`, `internal/cli/cli.go` | `main` stays thin; behavior is tested through `cli.Run(args, stdout, stderr, deps)` |
| Pull/push command flow | `internal/cli/command_pull.go`, `internal/cli/command_push.go`, `internal/cli/command_batch.go`, `internal/cli/runtime.go` | Load config, select targets/preflight, then open provider/service |
| List command flow | `internal/cli/command_list.go` | Uses project-only config; validates regex/type before provider open; prints metadata only |
| Target selection | `internal/cli/selection/selector.go` | `--all` sorts by secret name; explicit names dedupe in caller order |
| Core sync behavior | `internal/secretsync/` | Owns latest-enabled reads, atomic writes, create-version, create-missing, partial batch failures |
| Manifest policy | `internal/config/config.go`, `internal/mapping/types.go` | Strict JSON; unknown fields/trailing data rejected; `mode` required |
| Secret safety policy | `internal/secretcontract/policy.go`, `internal/pathpolicy/pathpolicy.go`, `internal/fsx/fsx.go` | `*-dev`, root-contained paths, atomic `0600` writes |
| Scaleway adapter | `internal/secretprovider/scaleway/api.go` | SDK request/response shaping; profile override; region parsing |
| Provider contracts | `internal/secretprovider/contracts/`, `internal/secretprovider/scaleway/contracts/`, `docs/contracts/provider-compatibility.md` | Live gate is read-only/list-only and build-tagged `integration` |
| Test fake | `internal/testdouble/secretapi/fake_secret_api.go` | Reuse for CLI/service tests; use deterministic constructor for stable IDs |
| Release/package flow | `.github/workflows/release.yml`, `.goreleaser.yaml`, `scripts/update-package-repos.sh` | Release Please -> GoReleaser -> optional Homebrew/Scoop sync |

## CODE MAP
| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `runMain` | function | `cmd/dev-vault/main.go` | Injects build metadata and real Scaleway provider into `cli.Run` |
| `cli.Run` | function | `internal/cli/cli.go` | Central testable CLI boundary; maps command errors to exit codes |
| `commandRuntime.prepareResources` | method | `internal/cli/runtime.go` | Loads config, opens provider, constructs `secretsync.Service` |
| `selection.SelectTargetsForMode` | function | `internal/cli/selection/selector.go` | Enforces `--all`/explicit target semantics and mapping modes |
| `secretsync.Service` | struct | `internal/secretsync/types.go` | Provider-independent runtime service for list/pull/push |
| `config.Load` / `LoadProject` | functions | `internal/config/config.go` | Full mapping validation for pull/push; project-only load for list |
| `secretprovider.SecretAPI` | interface | `internal/secretprovider/types.go` | Boundary between sync logic and provider implementations |
| `scaleway.Open` | function | `internal/secretprovider/scaleway/api.go` | Production Scaleway SDK opener |
| `secretcontract.ValidateDevSecretName` | function | `internal/secretcontract/policy.go` | Hard `-dev` invariant |
| `fsx.AtomicWriteFile` | function | `internal/fsx/fsx.go` | Atomic write helper used by pull |

## HARD SAFETY RULES
- Never print secret payloads to stdout, stderr, logs, test failures, or errors. Only metadata such as name/type/path/id/revision is allowed.
- Refuse any secret name that does not end with `-dev`; this is an invariant, not a convention.
- Do not commit credentials, tokens, real secret identifiers, tenant data, or live project IDs. Use sanitized examples.
- If a secret/token is pasted into chat/logs, treat it as compromised and rotate/revoke immediately.
- Live provider contracts must stay read-only/list-only: no payload access, mutation, create, push, or version reads.

## `.scw.json` CONTRACT
- File name is fixed: `.scw.json`; discovery searches upward unless `--config <path>` is passed.
- Project root is the directory containing `.scw.json`; `mapping[*].file` is relative to that root and must not escape it.
- Unknown JSON fields and trailing JSON data are rejected.
- `mapping` is required for pull/push and optional for list/project-only config.
- Mapping keys are Scaleway secret names and must end with `-dev`.
- `mapping[*].mode` is required: `pull`, `push`, or `skip`.
- `pull --all` selects only `mode=pull`; `push --all` selects only `mode=push`; `skip` is excluded from both.
- `mapping[*].format` defaults to `raw`; `dotenv` expects JSON object string values on pull and uploads JSON from `.env` on push.
- `push --create-missing` requires `mapping.type` and uses `mapping.path` default `/`.

## TESTING CONVENTIONS
- `make test` is the default gate and fails unless coverage is exactly `100.0%`.
- Unit tests do not call live Scaleway. Use `internal/testdouble/secretapi` for CLI/service behavior.
- Keep filesystem behavior real where practical: tests use `t.TempDir`, `os.WriteFile`, `os.ReadFile`, and verify atomic write/permission behavior.
- Provider-sensitive behavior belongs in SDK request-shaping tests plus the read-only provider contract suite; do not mock away provider drift completely.
- Repo contracts in `internal/repocontracts` assert workflow invariants, including `actions/setup-go@v6` with `go-version-file: go.mod`.

## COMMANDS
```bash
go run ./cmd/dev-vault -h
make test
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -n 1
ALLOW_CONTRACT_SKIP=1 make test-contracts
DEV_VAULT_TEST_PROJECT_ID=<project-id> DEV_VAULT_TEST_ORGANIZATION_ID=<org-id> DEV_VAULT_TEST_REGION=fr-par scripts/test-provider-contract.sh
act -W .github/workflows/ci.yml -j test
act -W .github/workflows/ci.yml -j build
```

## CI / RELEASE NOTES
- CI runs on `pull_request` and `push` to `main` only.
- CI gates: latest gitleaks download with `--redact`, exact 100% coverage, focused Scaleway adapter tests, optional live contracts with `ALLOW_CONTRACT_SKIP=1`, and multi-arch build smoke.
- GoReleaser builds `linux/darwin/windows` for `amd64/arm64` except `windows/arm64`, with `CGO_ENABLED=0` and ldflags for `main.version`, `main.commit`, `main.date`.
- Release Please creates the `v*` tag/release; publish then runs gitleaks, coverage, provider gate, GoReleaser, and best-effort Homebrew/Scoop sync.
- Tap sync needs `HOMEBREW_TAP_GITHUB_TOKEN`; missing token skips cleanly.
- Renovate automerges safe updates when CI is green; keep workflow `push` triggers limited to `main` to avoid branch noise.

## COMMIT RULES
- Conventional Commits are required: `<type>(optional-scope): <summary>`.
- Accepted types include `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.

<!-- desloppify-begin -->
<!-- desloppify-skill-version: 2 -->
---
name: desloppify
description: >
  Codebase health scanner and technical debt tracker. Use when the user asks
  about code quality, technical debt, dead code, large files, god classes,
  duplicate functions, code smells, naming issues, import cycles, or coupling
  problems. Also use when asked for a health score, what to fix next, or to
  create a cleanup plan. Supports 28 languages.
allowed-tools: Bash(desloppify *)
---

# Desloppify

## 1. Your Job

Improve code quality by maximising the **strict score** honestly.

**The main thing you do is run `desloppify next`** — it tells you exactly what to fix and how. Fix it, resolve it, run `next` again. Keep going.

Follow the scan output's **INSTRUCTIONS FOR AGENTS** — don't substitute your own analysis.

## 2. The Workflow

Two loops. The **outer loop** rescans periodically to measure progress.
The **inner loop** is where you spend most of your time: fixing issues one by one.

### Outer loop — scan and check

```bash
desloppify scan --path .       # analyse the codebase
desloppify status              # check scores — are we at target?
```
If not at target, work the inner loop. Rescan periodically — especially after clearing a cluster or batch of related fixes. Issues cascade-resolve and new ones may surface.

### Inner loop — fix issues

Repeat until the queue is clear:

```
1. desloppify next              ← tells you exactly what to fix next
2. Fix the issue in code
3. Resolve it (next shows you the exact command including required attestation)
```

Score may temporarily drop after fixes — cascade effects are normal, keep going.
If `next` suggests an auto-fixer, run `desloppify fix <fixer> --dry-run` to preview, then apply.

**To be strategic**, use `plan` to shape what `next` gives you:
```bash
desloppify plan                        # see the full ordered queue
desloppify plan move <pat> top         # reorder — what unblocks the most?
desloppify plan cluster create <name>  # group related issues to batch-fix
desloppify plan focus <cluster>        # scope next to one cluster
desloppify plan defer <pat>            # push low-value items aside
desloppify plan skip <pat>             # hide from next
desloppify plan done <pat>             # mark complete
desloppify plan reopen <pat>           # reopen
```

### Subjective reviews

The scan will prompt you when a subjective review is needed — just follow its instructions.
If you need to trigger one manually:
```bash
desloppify review --run-batches --runner codex --parallel --scan-after-import
```

### Other useful commands

```bash
desloppify next --count 5                         # top 5 priorities
desloppify next --cluster <name>                  # drill into a cluster
desloppify show <pattern>                         # filter by file/detector/ID
desloppify show --status open                     # all open findings
desloppify plan skip --permanent "<id>" --note "reason" # accept debt (lowers strict score)
desloppify scan --path . --reset-subjective       # reset subjective baseline to 0
```

## 3. Reference

### How scoring works

Overall score = **40% mechanical** + **60% subjective**.

- **Mechanical (40%)**: auto-detected issues — duplication, dead code, smells, unused imports, security. Fixed by changing code and rescanning.
- **Subjective (60%)**: design quality review — naming, error handling, abstractions, clarity. Starts at **0%** until reviewed. The scan will prompt you when a review is needed.
- **Strict score** is the north star: wontfix items count as open. The gap between overall and strict is your wontfix debt.
- **Score types**: overall (lenient), strict (wontfix counts), objective (mechanical only), verified (confirmed fixes only).

### Subjective reviews in detail

- **Preferred**: `desloppify review --run-batches --runner codex --parallel --scan-after-import` — does everything in one command.
- **Manual path**: `desloppify review --prepare` → review per dimension → `desloppify review --import file.json`.
- Import first, fix after — import creates tracked state entries for correlation.
- Integrity: reviewers score from evidence only. Scores hitting exact targets trigger auto-reset.
- Even moderate scores (60-80) dramatically improve overall health.
- Stale dimensions auto-surface in `next` — just follow the queue.

### Key concepts

- **Tiers**: T1 auto-fix → T2 quick manual → T3 judgment call → T4 major refactor.
- **Auto-clusters**: related findings are auto-grouped in `next`. Drill in with `next --cluster <name>`.
- **Zones**: production/script (scored), test/config/generated/vendor (not scored). Fix with `zone set`.
- **Wontfix cost**: widens the lenient↔strict gap. Challenge past decisions when the gap grows.
- Score can temporarily drop after fixes (cascade effects are normal).

## 4. Escalate Tool Issues Upstream

When desloppify itself appears wrong or inconsistent:

1. Capture a minimal repro (`command`, `path`, `expected`, `actual`).
2. Open a GitHub issue in `peteromallet/desloppify`.
3. If you can fix it safely, open a PR linked to that issue.
4. If unsure whether it is tool bug vs user workflow, issue first, PR second.

## Prerequisite

`command -v desloppify >/dev/null 2>&1 && echo "desloppify: installed" || echo "NOT INSTALLED — run: pip install --upgrade git+https://github.com/peteromallet/desloppify.git"`

<!-- desloppify-end -->

## Codex Overlay

This is the canonical Codex overlay used by the README install command.

1. Prefer first-class batch runs: `desloppify review --run-batches --runner codex --parallel --scan-after-import`.
2. The command writes immutable packet snapshots under `.desloppify/review_packets/holistic_packet_*.json`; use those for reproducible retries.
3. Keep reviewer input scoped to the immutable packet and the source files named in each batch.
4. Do not use prior chat context, score history, narrative summaries, issue labels, or target-threshold anchoring while scoring.
5. Assess every dimension listed in `query.dimensions`; never drop a requested dimension. If evidence is weak/mixed, score lower and explain uncertainty in findings.
6. Return machine-readable JSON only for review imports. For Claude session submit (`--external-submit`), include `session` from the generated template:

```json
{
  "session": {
    "id": "<session_id_from_template>",
    "token": "<session_token_from_template>"
  },
  "assessments": {
    "<dimension_from_query>": 0
  },
  "findings": [
    {
      "dimension": "<dimension_from_query>",
      "identifier": "short_id",
      "summary": "one-line defect summary",
      "related_files": ["relative/path/to/file.py"],
      "evidence": ["specific code observation"],
      "suggestion": "concrete fix recommendation",
      "confidence": "high|medium|low"
    }
  ]
}
```

7. `findings` MUST match `query.system_prompt` exactly (including `related_files`, `evidence`, and `suggestion`). Use `"findings": []` when no defects are found.
8. Import is fail-closed by default: if any finding is invalid/skipped, `desloppify review --import` aborts unless `--allow-partial` is explicitly passed.
9. Assessment scores are auto-applied from trusted internal run-batches imports, or via Claude cloud session imports (`desloppify review --external-start --external-runner claude` then printed `--external-submit`). Legacy attested external import via `--attested-external` remains supported.
10. Manual override is safety-scoped: you cannot combine it with `--allow-partial`, and provisional manual scores expire on the next `scan` unless replaced by trusted internal or attested-external imports.
11. If a batch fails, retry only that slice with `desloppify review --run-batches --packet <packet.json> --only-batches <idxs>`.

<!-- desloppify-overlay: codex -->
<!-- desloppify-end -->
