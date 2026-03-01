# Provider Compatibility Contract

This project keeps provider compatibility checks at the `internal/secretprovider` boundary.

## Contract Layers

- CLI boundary contract checks:
  - `internal/cli/contracts/batch_reporting_test.go`
  - Validates runtime boundary behavior for open-api failure propagation and preflight short-circuiting.
- DTO and request-shaping contract tests (offline):
  - `internal/secretprovider/scaleway/api_test.go`
  - `internal/secretprovider/scaleway/open_profile_test.go`
- Read-only live integration contract (optional):
  - `internal/secretprovider/scaleway/contracts/api_integration_test.go`
  - Runs list-based conformance checks (typed filtering, name filtering, invalid type error shape)
  - Never reads or prints secret payloads
  - Never mutates or creates secrets

## Running The Live Contract Gate

```bash
DEV_VAULT_TEST_PROJECT_ID=<project-id> \
DEV_VAULT_TEST_ORGANIZATION_ID=<org-id> \
DEV_VAULT_TEST_REGION=fr-par \
scripts/test-provider-contract.sh
```

The gate is skipped automatically when required env vars are not set.
