#!/usr/bin/env bash
set -euo pipefail

cd /mnt/d/AI/Local/Agent/gateway-go
source /etc/profile.d/sheathed-edge-env.sh
go run ./cmd/gateway -config /mnt/d/AI/Local/Agent/gateway.config.json
