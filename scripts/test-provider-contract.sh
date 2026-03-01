#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

required_env=(DEV_VAULT_TEST_PROJECT_ID DEV_VAULT_TEST_ORGANIZATION_ID)
missing=()
for name in "${required_env[@]}"; do
  if [ -z "${!name:-}" ]; then
    missing+=("$name")
  fi
done

if [ ${#missing[@]} -gt 0 ]; then
  if [ "${ALLOW_CONTRACT_SKIP:-0}" = "1" ]; then
    echo "provider contracts skipped (ALLOW_CONTRACT_SKIP=1, missing: ${missing[*]})"
    exit 0
  fi
  echo "missing required env vars for provider contracts: ${missing[*]}" >&2
  echo "set ALLOW_CONTRACT_SKIP=1 to skip explicitly" >&2
  exit 2
fi

go test ./internal/secretprovider/scaleway/contracts -tags=integration -run TestScalewayProviderReadOnlyContracts "$@"
