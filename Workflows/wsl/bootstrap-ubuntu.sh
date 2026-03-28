#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "bootstrap-ubuntu.sh 必须以 root 身份运行" >&2
  exit 1
fi

LINUX_USER="${1:-awdp}"
PROJECT_ROOT_WSL="${2:-/mnt/d/AI/Local}"
GO_VERSION="${GO_VERSION:-go1.26.1}"
UBUNTU_MIRROR="${UBUNTU_MIRROR:-https://mirrors.aliyun.com/ubuntu/}"
GO_DOWNLOAD_BASE="${GO_DOWNLOAD_BASE:-https://mirrors.aliyun.com/golang}"
PROFILE_SNIPPET="/etc/profile.d/sheathed-edge-env.sh"
WSL_CONF="/etc/wsl.conf"
APT_SOURCES_FILE="/etc/apt/sources.list"
APT_RETRY_FILE="/etc/apt/apt.conf.d/99sheathed-edge-retries"

echo "==> 配置 Ubuntu 镜像源"
rm -f /etc/apt/sources.list.d/ubuntu.sources
cat > "${APT_SOURCES_FILE}" <<EOF
deb ${UBUNTU_MIRROR} noble main universe restricted multiverse
deb ${UBUNTU_MIRROR} noble-updates main universe restricted multiverse
deb ${UBUNTU_MIRROR} noble-backports main universe restricted multiverse
deb ${UBUNTU_MIRROR} noble-security main universe restricted multiverse
EOF

cat > "${APT_RETRY_FILE}" <<'EOF'
Acquire::Retries "5";
Acquire::http::Timeout "20";
Acquire::https::Timeout "20";
Acquire::ForceIPv4 "true";
EOF

echo "==> apt 基线更新"
apt-get clean
rm -rf /var/lib/apt/lists/*
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  git \
  curl \
  wget \
  ca-certificates \
  build-essential \
  pkg-config \
  cmake \
  ninja-build \
  jq \
  unzip \
  zip \
  zstd \
  sqlite3 \
  libsqlite3-dev \
  ripgrep \
  fd-find \
  tmux \
  python3 \
  python3-venv \
  python3-pip \
  pipx

if ! id -u "${LINUX_USER}" >/dev/null 2>&1; then
  echo "==> 创建 Linux 用户 ${LINUX_USER}"
  useradd -m -s /bin/bash "${LINUX_USER}"
  usermod -aG sudo "${LINUX_USER}"
  echo "${LINUX_USER} ALL=(ALL) NOPASSWD:ALL" > "/etc/sudoers.d/90-${LINUX_USER}"
  chmod 0440 "/etc/sudoers.d/90-${LINUX_USER}"
fi

echo "==> 写入 /etc/wsl.conf"
cat > "${WSL_CONF}" <<EOF
[boot]
systemd=true

[automount]
enabled=true
mountFsTab=false
options=metadata,uid=1000,gid=1000,umask=022,fmask=011

[interop]
appendWindowsPath=false

[user]
default=${LINUX_USER}
EOF

echo "==> 创建缓存和工具目录"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.cache"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.cache/sheathed-edge"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.cache/go-build"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.cache/uv"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.venvs"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/go"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.cargo"
install -d -o "${LINUX_USER}" -g "${LINUX_USER}" "/home/${LINUX_USER}/.local/bin"
chown -R "${LINUX_USER}:${LINUX_USER}" "/home/${LINUX_USER}/.cache" "/home/${LINUX_USER}/.venvs" "/home/${LINUX_USER}/go" "/home/${LINUX_USER}/.cargo" "/home/${LINUX_USER}/.local"

echo "==> 写入环境变量"
cat > "${PROFILE_SNIPPET}" <<'EOF'
export CARGO_TARGET_DIR="$HOME/.cache/sheathed-edge/cargo-target"
export GOCACHE="$HOME/.cache/go-build"
export GOPATH="$HOME/go"
export GOMODCACHE="$HOME/go/pkg/mod"
export UV_CACHE_DIR="$HOME/.cache/uv"
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:/usr/local/go/bin:$PATH"
EOF

chmod 0644 "${PROFILE_SNIPPET}"

echo "==> 安装 uv"
if su - "${LINUX_USER}" -c "command -v uv >/dev/null 2>&1"; then
  echo "uv already installed, skipping"
else
  su - "${LINUX_USER}" -c "curl -LsSf https://astral.sh/uv/install.sh | sh"
fi

echo "==> 安装 Rust stable"
if su - "${LINUX_USER}" -c "command -v rustup >/dev/null 2>&1 && command -v cargo >/dev/null 2>&1"; then
  echo "Rust toolchain already installed, skipping"
else
  su - "${LINUX_USER}" -c "curl https://sh.rustup.rs -sSf | sh -s -- -y --profile default --default-toolchain stable"
fi

echo "==> 安装 Go ${GO_VERSION}"
if [[ -x /usr/local/go/bin/go ]] && /usr/local/go/bin/go version | grep -q "${GO_VERSION}"; then
  echo "Go ${GO_VERSION} already installed, skipping"
else
  curl -fsSL "${GO_DOWNLOAD_BASE}/${GO_VERSION}.linux-amd64.tar.gz" -o "/tmp/${GO_VERSION}.linux-amd64.tar.gz"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_VERSION}.linux-amd64.tar.gz"
  rm -f "/tmp/${GO_VERSION}.linux-amd64.tar.gz"
fi

echo "==> 配置 Git"
su - "${LINUX_USER}" -c "git config --global core.autocrlf input"
su - "${LINUX_USER}" -c "git config --global core.filemode false"
su - "${LINUX_USER}" -c "git config --global init.defaultBranch main"

echo "==> 为项目目录启用大小写敏感"
if command -v fsutil.exe >/dev/null 2>&1; then
  fsutil.exe file setCaseSensitiveInfo "$(wslpath -w "${PROJECT_ROOT_WSL}")" enable >/dev/null 2>&1 || true
fi

echo "==> 初始化完成"
echo "请在 Windows 侧执行: wsl.exe --terminate Ubuntu-24.04"
