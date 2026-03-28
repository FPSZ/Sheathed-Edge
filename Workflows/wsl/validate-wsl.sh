#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT_WSL="${1:-/mnt/d/AI/Local}"
LLAMA_DEFAULT_URL="${LLAMA_BASE_URL:-http://127.0.0.1:8080}"

if [[ -f /etc/profile.d/sheathed-edge-env.sh ]]; then
  # Load toolchain PATH and cache vars written during bootstrap.
  # shellcheck disable=SC1091
  source /etc/profile.d/sheathed-edge-env.sh
fi

pass() {
  printf '[PASS] %s\n' "$1"
}

warn() {
  printf '[WARN] %s\n' "$1"
}

fail() {
  printf '[FAIL] %s\n' "$1" >&2
  exit 1
}

check_cmd() {
  local cmd="$1"
  command -v "${cmd}" >/dev/null 2>&1 || fail "缺少命令: ${cmd}"
  pass "命令存在: ${cmd}"
}

echo "==> systemd"
systemd_state="$(systemctl is-system-running 2>/dev/null || true)"
if [[ "${systemd_state}" == "running" || "${systemd_state}" == "degraded" ]]; then
  pass "systemd 状态可接受: ${systemd_state}"
else
  warn "systemd 当前状态: ${systemd_state:-unknown}"
fi

echo "==> 项目挂载"
[[ -d "${PROJECT_ROOT_WSL}" ]] || fail "项目目录不存在: ${PROJECT_ROOT_WSL}"
[[ -w "${PROJECT_ROOT_WSL}" ]] || fail "项目目录不可写: ${PROJECT_ROOT_WSL}"
pass "项目目录可读写: ${PROJECT_ROOT_WSL}"

echo "==> 工具链"
for cmd in python3 cargo rustc go sqlite3 rg git curl; do
  check_cmd "${cmd}"
done

echo "==> Linux 原生路径"
for cmd in python3 cargo rustc go git; do
  resolved="$(readlink -f "$(command -v "${cmd}")")"
  if [[ "${resolved}" == /mnt/* ]] || [[ "${resolved}" == *.exe ]]; then
    fail "${cmd} 指向了非 Linux 原生路径: ${resolved}"
  fi
  pass "${cmd} 路径正常: ${resolved}"
done

echo "==> 缓存变量"
[[ "${CARGO_TARGET_DIR:-}" == "$HOME/.cache/sheathed-edge/cargo-target" ]] || warn "CARGO_TARGET_DIR 未生效"
[[ "${GOCACHE:-}" == "$HOME/.cache/go-build" ]] || warn "GOCACHE 未生效"
[[ "${GOPATH:-}" == "$HOME/go" ]] || warn "GOPATH 未生效"
[[ "${GOMODCACHE:-}" == "$HOME/go/pkg/mod" ]] || warn "GOMODCACHE 未生效"
[[ "${UV_CACHE_DIR:-}" == "$HOME/.cache/uv" ]] || warn "UV_CACHE_DIR 未生效"

echo "==> llama-server 可达性"
if curl -fsS --max-time 3 "${LLAMA_DEFAULT_URL}/health" >/dev/null 2>&1; then
  pass "默认地址可达: ${LLAMA_DEFAULT_URL}"
else
  HOST_IP="$(ip route show default 2>/dev/null | awk '/default/ {print $3; exit}')"
  if [[ -n "${HOST_IP}" ]] && curl -fsS --max-time 3 "http://${HOST_IP}:8080/health" >/dev/null 2>&1; then
    pass "默认路由宿主机地址可达: http://${HOST_IP}:8080"
  else
    warn "未探测到可用的 llama-server，请确认 Windows 侧已启动"
  fi
fi

echo "==> 验证完成"
