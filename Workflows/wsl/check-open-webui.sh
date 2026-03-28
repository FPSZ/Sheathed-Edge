#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:3000}"

echo "Checking ${BASE_URL}/health"
curl -fsS "${BASE_URL}/health" || curl -fsS "${BASE_URL}/healthz"
