# Agent 路由与 Skills 分层实施说明

## 本次更新做了什么

这次更新把 `pwn + web` 的解题入口收成了四层：

1. 共享总入口 `agent.md`
2. `pwn` / `web` 各自的 prompt fragment
3. `pwn` / `web` 各自的大类 skills 文档
4. `Agent Layer Presets` 控制层

目标是让模型先做题型路由，再进入对应 family skills，同时把这些层做成可管理、可组合、可预览的对象。

当前交付状态：

- `Phase A` 已完成：
  - Gateway Admin 已支持 `Agent Layers` 预设管理
  - Gateway 后端已支持 `metadata.agent_layers`
  - 会话日志与阶段日志已记录 `agent_layers`
- `Phase B` 已预留：
  - Open WebUI 聊天页可以后续直接复用 `metadata.agent_layers`
  - 但本仓库里没有 Open WebUI 聊天前端源码，所以这一步暂未在聊天页落按钮

## 现在的入口关系

### 1. 共享总入口

主文件：

- `D:\AI\Local\Core\awdp\prompts\agent.md`

职责：

- 定义统一的 `task_family` 路由
- 定义 `phase / primary_skill / evidence_required / fallback_if_fail`
- 约束“先分类、再选路线、再执行”

如果后续要改总 Agent 行为，优先改这个文件。

### 2. Pwn 插件层

文件：

- `D:\AI\Local\Plugins\pwn\plugin.json`
- `D:\AI\Local\Plugins\pwn\prompts\fragment.md`
- `D:\AI\Local\Plugins\pwn\skills\pwn-skills.md`

职责分工：

- `plugin.json`
  - 负责挂载关系
  - 不负责写长篇规则
- `prompts/fragment.md`
  - 负责补充 `pwn` 的路由纪律、工具纪律、禁止项
- `skills/pwn-skills.md`
  - 负责 `pwn` family 的阶段化技能内容

### 3. Web 插件层

文件：

- `D:\AI\Local\Plugins\web\plugin.json`
- `D:\AI\Local\Plugins\web\prompts\fragment.md`
- `D:\AI\Local\Plugins\web\skills\web-skills.md`

职责分工：

- `plugin.json`
  - 负责挂载关系
  - 不负责写长篇规则
- `prompts/fragment.md`
  - 负责补充 `web` 的路由纪律、工具纪律、禁止项
- `skills/web-skills.md`
  - 负责 `web` family 的阶段化技能内容

### 4. Agent Layers 控制层

配置文件：

- `D:\AI\Local\Agent\agent-layer-presets.json`

后端实现：

- `D:\AI\Local\Agent\gateway-go\internal\gateway\admin\agent_layers.go`
- `D:\AI\Local\Agent\gateway-go\internal\gateway\orchestrator\orchestrator.go`
- `D:\AI\Local\Agent\gateway-go\internal\gateway\mode\loader.go`

Admin 页面：

- `D:\AI\Local\ui\gateway-admin\src\pages\AgentLayersPage.tsx`
- `D:\AI\Local\ui\gateway-admin\src\app\router.tsx`
- `D:\AI\Local\ui\gateway-admin\src\app\AppShell.tsx`
- `D:\AI\Local\ui\gateway-admin\src\components\Sidebar.tsx`

职责：

- 管理 `agent router / pwn skills / web skills` 三层开关
- 提供预设，例如：
  - `router-only`
  - `router-pwn`
  - `router-web`
  - `router-pwn-web`
- 预览每个预设最终会挂载哪些：
  - `effective_prompt_files`
  - `effective_skill_files`
  - `effective_tool_scope`
  - `effective_retrieval_roots`

## 后续 AI 该改哪里

### 要改总路由逻辑

改：

- `D:\AI\Local\Core\awdp\prompts\agent.md`

不要先改：

- `pwn-skills.md`
- `web-skills.md`

因为题型路由属于共享入口，不属于某个 family 技能。

### 要改 pwn 专属行为

改：

- `D:\AI\Local\Plugins\pwn\prompts\fragment.md`
- `D:\AI\Local\Plugins\pwn\skills\pwn-skills.md`

规则：

- 改 `fragment.md`：改 `pwn` 插件启用时的纪律与提示
- 改 `pwn-skills.md`：改 `pwn` 家族具体技能内容

### 要改 web 专属行为

改：

- `D:\AI\Local\Plugins\web\prompts\fragment.md`
- `D:\AI\Local\Plugins\web\skills\web-skills.md`

规则：

- 改 `fragment.md`：改 `web` 插件启用时的纪律与提示
- 改 `web-skills.md`：改 `web` 家族具体技能内容

### 要改挂载关系

改：

- `D:\AI\Local\Plugins\pwn\plugin.json`
- `D:\AI\Local\Plugins\web\plugin.json`

这里只负责：

- `prompt_files`
- `skill_files`
- `tool_scope`
- `retrieval_roots`

不要把长篇知识直接写进 `plugin.json`。

### 要改哪些层默认开、有哪些预设

改：

- `D:\AI\Local\Agent\agent-layer-presets.json`

如果是改后端预览逻辑或 Admin 管理逻辑，再看：

- `D:\AI\Local\Agent\gateway-go\internal\gateway\admin\agent_layers.go`
- `D:\AI\Local\ui\gateway-admin\src\pages\AgentLayersPage.tsx`

如果是改会话级承接逻辑，再看：

- `D:\AI\Local\Agent\gateway-go\internal\gateway\orchestrator\orchestrator.go`
- `D:\AI\Local\Agent\gateway-go\internal\gateway\mode\loader.go`
- `D:\AI\Local\Agent\gateway-go\internal\gateway\logging\session.go`
- `D:\AI\Local\Agent\gateway-go\internal\gateway\logging\stage.go`

## 当前固定术语

后续训练、评测、技能迭代默认沿用这些名字：

- `task_family`
- `phase`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

其中：

- `task_family` 第一版只用：
  - `pwn`
  - `web`
  - `uncertain`
- `phase` 第一版只用：
  - `intake`
  - `triage`
  - `hypothesis`
  - `verification`
  - `exploit_or_patch`
  - `finalization`

## 当前边界

这次已经做了：

- `pwn + web` 的统一路由入口
- family prompt / skills 分层
- Agent Layers 预设控制层
- 后端对 `metadata.agent_layers` 的会话承接
- Admin 预设与预览页面
- `Gateway Admin` 新增 `/admin/agent-layers` 页面入口
- 预设默认文件落盘到 `D:\AI\Local\Agent\agent-layer-presets.json`

这次还没做：

- `reverse` 并入主路由
- 正式训练样本转换
- Open WebUI 聊天页按钮接入
- 自动从线上日志回灌训练

## WebUI 后续怎么接

后续如果要把开关真正做进 WebUI 会话页，优先遵守这条最小链路：

1. 聊天页只传会话级选择，不直接修改全局 md 或 preset 文件。
2. 优先沿用现有 `metadata`，在请求里补：
   - `metadata.agent_layers.enable_agent_router`
   - `metadata.agent_layers.enable_pwn_skills`
   - `metadata.agent_layers.enable_web_skills`
3. 如果改成“选预设”而不是“逐项开关”，则在 WebUI 侧先把 preset 展开成上面三个布尔值，再发请求。
4. 不要让 WebUI 直接编辑 `plugin.json`、`agent.md`、`pwn-skills.md`、`web-skills.md`。

这样做的原因是：

- 会话选择只影响当前会话
- 文档维护和 UI 开关解耦
- 后续训练能直接看到结构化 layer 组合

## 维护注意事项

- `agent.md` 是共享路由入口，不要塞成知识大全。
- `pwn-skills.md` 和 `web-skills.md` 才是长大的知识层。
- `agent-layer-presets.json` 只负责开关组合，不负责写知识。
- 如果只是补一个具体题型经验，优先补到对应 family skills。
- 如果只是调“默认开哪层、哪组预设给会话用”，优先改 `agent-layer-presets.json`。
- 如果只是调会话时启用哪层，不要改文档，直接走 `metadata.agent_layers`。
