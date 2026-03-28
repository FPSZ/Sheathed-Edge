#!/usr/bin/env bash
set -euo pipefail

source /etc/profile.d/sheathed-edge-env.sh

VENV_DIR="${OPEN_WEBUI_VENV:-$HOME/.venvs/open-webui}"
PYTHON_BIN="${OPEN_WEBUI_PYTHON:-python3}"

mkdir -p "$(dirname "$VENV_DIR")" "$HOME/.cache/sheathed-edge/open-webui"

if [[ ! -d "$VENV_DIR" ]]; then
  "$PYTHON_BIN" -m venv "$VENV_DIR"
fi

source "$VENV_DIR/bin/activate"
python -m pip install --upgrade pip
python -m pip install "open-webui==0.8.12"

echo "Open WebUI installed in $VENV_DIR"
echo "Run: /mnt/d/AI/Local/Workflows/wsl/start-open-webui.sh"
