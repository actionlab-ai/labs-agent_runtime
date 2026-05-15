#!/usr/bin/env bash
set -euo pipefail

# Checks that core runtime packages do not import Google/GCP service SDKs or
# optional Google adapter packages. The repo still uses genai value types as the
# neutral content schema in some core APIs; this check is focused on the
# provider-capability boundary introduced by adapters/google.
roots=(agent artifact internal memory model plugin runner server session telemetry tool util)
pattern='"(cloud\.google\.com/|google\.golang\.org/api|google\.golang\.org/adk/adapters/google/)'

matches=$(rg -n --glob '*.go' --glob '!**/*_test.go' --glob '!internal/testutil/**' "$pattern" "${roots[@]}" || true)
if [[ -n "$matches" ]]; then
  echo "Core runtime imports Google/GCP service SDKs or Google adapters:" >&2
  echo "$matches" >&2
  exit 1
fi
