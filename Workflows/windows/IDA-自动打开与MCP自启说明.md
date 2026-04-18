# IDA 自动打开 + MCP 自动桥接说明

本目录新增两个脚本，目标是让 AI 或人工都能用一条命令完成：
- 打开指定题目到 IDA 9.1
- 自动启动 `mcp-plugin.py` 的 HTTP bridge
- 等待 `http://127.0.0.1:13337/mcp` 就绪

## 文件

- `D:\AI\Local\Workflows\windows\ida-open-and-wait.py`
  - Windows 侧启动器
  - 负责启动 `ida.exe`、传递环境变量、轮询 bridge
- `D:\AI\Local\Workflows\windows\ida-mcp-bootstrap.py`
  - IDAPython 启动脚本
  - 由 IDA 命令行 `-S` 自动执行
  - 在 IDA 进程内加载 `plugins\mcp-plugin.py` 并直接启动 `Server()`

## 设计要点

当前本机 `plugins\mcp-plugin.py` 的原始行为是：
- 插件加载后只打印提示
- 必须人工在 IDA 菜单里点 `Edit -> Plugins -> MCP`
- 或按热键 `Ctrl-Alt-M`
- 才会调用 `run()` 启动 bridge

本方案不要求人工再点一次。
启动器会用：
- `ida.exe -S<bootstrap.py> <target>`

而 `ida-mcp-bootstrap.py` 会在 IDA 内：
- 读取环境变量里的 host / port / plugin path
- 动态加载 `mcp-plugin.py`
- 直接实例化并启动 `Server()`
- 把 server 对象挂到 `builtins.__sheathed_edge_ida_mcp_server`
  - 避免脚本退出后对象被回收

## 默认路径

- IDA: `D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\ida.exe`
- MCP 插件: `D:\CTF\tool\SURBIFByb2Zlc3Npb25hbCA5LjE\plugins\mcp-plugin.py`
- Bridge: `http://127.0.0.1:13337/mcp`

## 使用方法

### 人工测试

```powershell
D:\Environment2\Create\Python314\python.exe D:\AI\Local\Workflows\windows\ida-open-and-wait.py D:\CTF\sample\test.exe --analysis
```

或：

```powershell
python D:\AI\Local\Workflows\windows\ida-open-and-wait.py C:\Windows\System32\notepad.exe --analysis
```

### 仅复用已存在 bridge

```powershell
python D:\AI\Local\Workflows\windows\ida-open-and-wait.py C:\Windows\System32\notepad.exe --allow-existing-bridge
```

## 返回结果

脚本输出 JSON，关键字段：
- `ok`
- `ida_launched`
- `bridge_ready`
- `existing_bridge`
- `bridge_url`
- `pid`
- `note`

## 自动替换旧会话

现在 `ida_open_file` / `ida-open-and-wait.py` 已支持“自动换题”。

行为如下：
- 如果 `13337` 上已经有旧的 IDA bridge
- 启动器会先关闭当前的 `ida.exe` / `idat.exe`
- 等旧 bridge 端口释放
- 再打开新题并重新拉起 bridge

这意味着 AI 在分析新题时，不需要先提醒用户手工关闭旧 IDA。
默认就会替换旧会话。

如果只想清场，不想立即开新题，可用：
- 适配器工具：`ida_close_active_session`
- 启动器参数：`--close-only`

## 已完成的正式工具接入

现在不只是脚本，`IDA` 已经接成了正式 MCP 适配器：

- 适配器文件：`D:\AI\Local\MCP\ida_bridge_adapter.py`
- MCP Server 配置项：`D:\AI\Local\Agent\mcp-servers.json` 里的 `ida-9-1-bridge`
- 当前暴露给 AI 的工具：
  - `ida_open_file`
  - `ida_bridge_status`
  - `ida_list_rpc_methods`
  - `ida_rpc_call`

也就是说，后面 WebUI / Gateway 里的 AI 不需要先人工开 IDA。
它可以先调：
- `ida_open_file`

再调：
- `ida_list_rpc_methods`
- `ida_rpc_call`

其中 `ida_rpc_call` 会把请求转给 IDA 进程内的原始 JSON-RPC bridge。

## 后续接入建议

如果后面要让 WebUI 里的 AI 直接调它，优先按下面顺序做：

### 方案 A：先通过现有 terminal / shell 能力调用
最省改动，适合立刻验证。
AI 先调用：
- `python D:\AI\Local\Workflows\windows\ida-open-and-wait.py <sample>`

bridge ready 后，再去使用：
- `ida-9-1-bridge`

### 方案 B：再把它包装成正式工具
建议新增一个本地工具层，例如：
- `ida_open_file`
- `ida_bridge_status`

让 host-agent 或 tool-router 暴露成可调用工具。
这样 AI 不需要自己拼命令。

## 如果后面要改这套逻辑，优先改哪里

### 改启动逻辑
- `D:\AI\Local\Workflows\windows\ida-open-and-wait.py`

### 改 IDA 内 bridge 自启逻辑
- `D:\AI\Local\Workflows\windows\ida-mcp-bootstrap.py`

### 改 IDA MCP 服务定义
- `D:\AI\Local\Agent\mcp-servers.json`
  - `ida-9-1-bridge`

### 改 reverse / binary 提示词里对 IDA 的使用说明
- `D:\AI\Local\Core\awdp\prompts\binary-core.md`
- `D:\AI\Local\Plugins\reverse\prompts\fragment.md`

## 备注

这一版先解决“自动打开 IDA + 自动拉起 bridge”。
它还没有自动刷新 tool-router 的 MCP discover 状态；如果要完全无感，需要下一步把它正式接成工具，并在启动成功后触发 MCP refresh。

## Typed IDA Tools

为了让小模型少拼 `ida_rpc_call`，现在又补了一批直接工具：

- `ida_get_metadata`
- `ida_list_functions`
- `ida_list_strings`
- `ida_get_function_by_name`
- `ida_get_function_by_address`
- `ida_decompile_function`
- `ida_disassemble_function`
- `ida_get_xrefs_to`
- `ida_get_callers`
- `ida_get_callees`
- `ida_close_active_session`

建议优先使用这些 typed tools，只有在没有对应 typed tool 时再退回 `ida_list_rpc_methods` / `ida_rpc_call`。

### 推荐给小模型的固定 reverse 开场

如果是新二进制题，优先让模型按这个顺序走：

- `ida_open_file`
- `ida_get_metadata`
- `ida_list_strings`
- `ida_list_functions`
- `ida_get_xrefs_to` / `ida_get_callers` / `ida_get_callees`
- `ida_decompile_function`

只在 typed tool 覆盖不到时，再退回：

- `ida_list_rpc_methods`
- `ida_rpc_call`
