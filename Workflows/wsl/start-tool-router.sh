#!/usr/bin/env bash
set -euo pipefail

cd /mnt/d/AI/Local/Agent/tool-router-rs
source /etc/profile.d/sheathed-edge-env.sh
cargo run -- --config /mnt/d/AI/Local/Agent/tool-router.config.json
