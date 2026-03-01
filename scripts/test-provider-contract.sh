#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

go test ./internal/secretprovider/scaleway -tags=integration -run TestScalewaySecretAPI_IntegrationListOpaque "$@"
