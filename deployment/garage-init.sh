#!/usr/bin/env bash
# One-time (idempotent) Garage setup: node layout, API key, bucket, website mode.
# Run after `make up`:  make garage-init
# Production: make prod-garage-init (COMPOSE_FILE and GARAGE_SERVICE select the stack).
set -euo pipefail
cd "$(dirname "$0")"
set -a; source .env; set +a

g() { docker compose exec -T -e RUST_LOG=warn "${GARAGE_SERVICE:-garage}" /garage "$@"; }

echo "waiting for garage..."
for _ in $(seq 1 30); do g status >/dev/null 2>&1 && break; sleep 1; done

if g status | grep -q "NO ROLE ASSIGNED"; then
  node_id=$(g node id -q | cut -d@ -f1)
  echo "assigning layout to node ${node_id:0:16}..."
  g layout assign -z dc1 -c 10G "$node_id"
  g layout apply --version 1
fi

if ! g key info "$S3_ACCESS_KEY_ID" >/dev/null 2>&1; then
  echo "importing API key..."
  g key import --yes -n restaurants-api "$S3_ACCESS_KEY_ID" "$S3_SECRET_ACCESS_KEY"
fi

if ! g bucket info "$S3_BUCKET" >/dev/null 2>&1; then
  echo "creating bucket $S3_BUCKET..."
  g bucket create "$S3_BUCKET"
fi
g bucket allow --read --write --owner "$S3_BUCKET" --key "$S3_ACCESS_KEY_ID" >/dev/null
g bucket website --allow "$S3_BUCKET" >/dev/null

echo "garage ready: bucket '$S3_BUCKET', public at http://$S3_BUCKET.web.garage.localhost:3902/"
