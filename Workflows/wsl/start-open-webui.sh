#!/usr/bin/env bash
set -euo pipefail

source /etc/profile.d/sheathed-edge-env.sh

VENV_DIR="${OPEN_WEBUI_VENV:-$HOME/.venvs/open-webui}"
DATA_DIR="${OPEN_WEBUI_DATA_DIR:-$HOME/.cache/sheathed-edge/open-webui-awdp}"
HOST="${OPEN_WEBUI_HOST:-127.0.0.1}"
PORT="${OPEN_WEBUI_PORT:-3000}"

if [[ ! -x "$VENV_DIR/bin/open-webui" ]]; then
  echo "open-webui is not installed in $VENV_DIR" >&2
  echo "Run: /mnt/d/AI/Local/Workflows/wsl/install-open-webui.sh" >&2
  exit 1
fi

mkdir -p "$DATA_DIR"

export DATA_DIR="$DATA_DIR"
export WEBUI_AUTH="${WEBUI_AUTH:-False}"
export OFFLINE_MODE="${OFFLINE_MODE:-True}"
export ENABLE_VERSION_UPDATE_CHECK="${ENABLE_VERSION_UPDATE_CHECK:-False}"
export BYPASS_EMBEDDING_AND_RETRIEVAL="${BYPASS_EMBEDDING_AND_RETRIEVAL:-True}"
export RAG_EMBEDDING_ENGINE="${RAG_EMBEDDING_ENGINE:-openai}"
export RAG_EMBEDDING_MODEL="${RAG_EMBEDDING_MODEL:-disabled-offline-embedding}"
export RAG_OPENAI_API_BASE_URL="${RAG_OPENAI_API_BASE_URL:-http://127.0.0.1:9/v1}"
export RAG_OPENAI_API_KEY="${RAG_OPENAI_API_KEY:-disabled}"
export RAG_EMBEDDING_MODEL_AUTO_UPDATE="${RAG_EMBEDDING_MODEL_AUTO_UPDATE:-False}"
export HF_HUB_OFFLINE="${HF_HUB_OFFLINE:-1}"

exec "$VENV_DIR/bin/open-webui" serve --host "$HOST" --port "$PORT"
