#!/usr/bin/env bash
# Create deployment/.env from .env.example, replacing each __RANDOM_HEX_N__
# placeholder with a fresh `openssl rand -hex N`. No-op if .env exists.
set -euo pipefail
cd "$(dirname "$0")"
if [[ -f .env ]]; then echo "deployment/.env exists, leaving it alone"; exit 0; fi
while IFS= read -r line; do
  while [[ $line =~ __RANDOM_HEX_([0-9]+)__ ]]; do
    line=${line/"${BASH_REMATCH[0]}"/$(openssl rand -hex "${BASH_REMATCH[1]}")}
  done
  printf '%s\n' "$line"
done < .env.example > .env
echo "created deployment/.env"
