# WSL2 环境落地说明

这份文档对应 `D:\AI\Local` 的 AWDP 本地系统基线，目标是把 WSL2 从“计划”变成“可重复执行的安装流程”。

## 已落地的文件

- `Workflows/wsl/install-wsl.ps1`
- `Workflows/wsl/bootstrap-ubuntu.sh`
- `Workflows/wsl/validate-wsl.sh`
- `Workflows/wsl/validate-wsl.ps1`

## 当前状态

截至 `2026-03-28`，这套 WSL2 基线已经完成落地并通过验收：

- `Ubuntu-24.04` 已导入并可正常启动
- `systemd` 已启用
- Linux 原生 `Go / Rust / Python / SQLite / ripgrep / Git` 已就绪
- WSL 工作目录已固定为 `/mnt/d/AI/Local`
- 唯一未闭环项是 `llama-server` 尚未在 Windows 侧启动，因此模型服务探测仍为告警

## 固定约束

- 发行版固定为 `Ubuntu-24.04`
- 安装位置固定为 `D:\Environment2\WSL`
- 仓库继续使用 `D:\AI\Local`，WSL 内映射为 `/mnt/d/AI/Local`
- 主模型推理继续在 Windows 宿主机，WSL 只跑服务层
- WSL 内只使用 Linux 原生 `Go / Rust / Python`

## 执行顺序

### 1. 以管理员 PowerShell 执行安装脚本

```powershell
Set-ExecutionPolicy -Scope Process Bypass
cd D:\AI\Local
.\Workflows\wsl\install-wsl.ps1
```

这一步会做两件事：

- 写入 `%UserProfile%\.wslconfig`
- 启用 `Microsoft-Windows-Subsystem-Linux` 和 `VirtualMachinePlatform`

脚本会在启用系统功能后直接退出，并要求重启。这是设计行为，不是失败。

如果重启后 `wsl.exe --status` 仍提示先运行 `wsl --install`，说明当前机器还缺少 WSL runtime。脚本会自动补跑：

```powershell
wsl.exe --install --no-distribution --web-download
```

这一步来自 Microsoft Learn 的 WSL 基础命令说明：当 WSL 尚未安装时，`--no-distribution` 可只安装 WSL 本体而不安装默认发行版，`--web-download` 可绕过 Microsoft Store，直接从在线源安装。

如果当前网络或策略会让 `web-download` / `inbox` 都返回 `403`，脚本会继续回退到官方 GitHub 发布的 WSL MSI，并调用 `msiexec` 本地安装。

当前这台机器的实际表现表明 `mirrored` 模式下 WSL 内公网访问不稳定，因此落地配置已经调整为 `networkingMode=nat`。原因是：

- `mirrored` 下 WSL 内所有 HTTPS 站点请求均超时
- 项目本身已经要求 Gateway 实现宿主机 IP fallback
- 对本项目来说，稳定联网和稳定构建优先级高于 `mirrored` 的理论便利性

### 2. 重启 Windows

系统功能启用后必须重启，否则 `wsl.exe --status` 仍会不可用。

### 3. 重启后继续导入和初始化

```powershell
Set-ExecutionPolicy -Scope Process Bypass
cd D:\AI\Local
.\Workflows\wsl\install-wsl.ps1 -SkipFeatureInstall
```

这一步会继续：

- 下载 Ubuntu 24.04 minimal rootfs
- 校验 `SHA256`
- 直接使用官方 `root.tar.xz` 导入到 `D:\Environment2\WSL\Distros\Ubuntu-24.04`
- 在 WSL 内执行初始化脚本

### 4. 验证环境

```powershell
cd D:\AI\Local
.\Workflows\wsl\validate-wsl.ps1
```

### 5. 导出快照

```powershell
wsl.exe --export Ubuntu-24.04 D:\Environment2\WSL\Export\ubuntu-24.04-baseline.tar
```

## 初始化脚本完成的内容

- 写入 `/etc/wsl.conf`
- 启用 `systemd`
- 创建默认用户 `awdp`
- 配置 `sudo` 免密
- 安装基础包、`uv`、`rustup stable`、`Go 1.26.1`
- 配置 Git 默认行为
- 创建缓存和工具目录
- 尝试为 `D:\AI\Local` 启用大小写敏感
- 将 Ubuntu 包源固定到国内可用镜像

## 当前已知限制

- Windows 本地不能先解包 Linux rootfs 再重打包，否则会破坏符号链接和设备节点；当前脚本已固定为直接导入官方归档
- `llama-server` 网络验收要等 Windows 侧模型服务启动后再做
