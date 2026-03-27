# 本地AI架构计划

## 1. 架构目标与约束

这份文档定义创13本机离线 AI 系统的固定架构，用于搭建一套围绕 `AWDP` 的本地作战系统。本文档不是部署脚本，也不是临时笔记，而是后续实施、迭代和验收的结构设计基线。

目标：

- 以本地离线推理为核心，支撑 AWDP 题目分析、日志归因、补丁建议、利用链判断
- 在 `awdp` 主模式下叠加 `web` / `pwn` 子插件，应对复合题
- 通过统一编排层和统一工具层实现可控、可审计、可扩展的工具调用
- 为后续外部训练、数据回灌和插件扩展预留稳定接口

硬约束：

- 本地只保留一个主模型
- 主模型固定为 `DeepSeek-R1-Distill-Llama-70B-Q4_K_M.gguf`
- 推理后端固定为 `llama.cpp / llama-server`
- 前端固定为 `Open WebUI`
- 工具协议统一为 `MCP`
- `awdp` 是唯一主模式，`web` / `pwn` 仅为可叠加子插件
- Windows 原生运行 `llama-server`
- `WSL2` 运行 `Open WebUI`、`Agent Gateway`、`Tool Router`、本地 MCP 服务
- V1 不做分布式、不做多用户、不依赖远端推理
- V1 不引入第二个常驻模型，不引入本地向量数据库

设计原则：

- 单主脑：所有最终推理、补丁结论、利用判断都由同一个主模型完成
- 控制面集中：所有模式切换、检索、工具路由、安全校验都归 `Agent Gateway + Tool Router`
- 工具最小暴露：按主模式和插件白名单开放工具
- 失败可回退：工具不可用时仍能回退到无工具推理
- 全链路可审计：会话、检索、工具、评测全部可还原

---

## 2. 总体组件图

```text
┌─────────────────────────────────────────────────────────┐
│                         用户 / 浏览器                    │
└────────────────────────────┬────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│                    Open WebUI  (WSL2)                   │
│  - 会话管理                                              │
│  - 文件上传                                              │
│  - 历史记录                                              │
│  - 逻辑模型别名 awdp-r1-70b                              │
└────────────────────────────┬────────────────────────────┘
                             │ OpenAI-compatible API
                             ▼
┌─────────────────────────────────────────────────────────┐
│               Agent Gateway / Orchestrator (WSL2)       │
│  - 注入 awdp 主提示词                                     │
│  - 激活 web / pwn 子插件                                  │
│  - 检索知识库                                             │
│  - 约束 JSON action envelope                             │
│  - 工具调用验证                                           │
│  - 审计与会话日志                                          │
└───────────────┬───────────────────────────┬─────────────┘
                │                           │
                │                           │
                ▼                           ▼
┌────────────────────────────┐   ┌────────────────────────┐
│ llama-server (Windows)     │   │ Tool Router (WSL2)     │
│ - 加载唯一 GGUF 模型         │   │ - 工具白名单             │
│ - OpenAI-compatible 接口     │   │ - 路由/校验/超时         │
│ - 仅负责推理                 │   │ - MCP 注册与日志         │
└────────────────────────────┘   └───────────┬────────────┘
                                              │
                                              ▼
                               ┌──────────────────────────┐
                               │ MCP Servers (WSL2)       │
                               │ - filesystem/search      │
                               │ - terminal               │
                               │ - retrieval              │
                               │ - browser                │
                               │ - radare2                │
                               │ - optional docker        │
                               └──────────────────────────┘
```

---

## 3. 组件职责分解

### 3.1 UI 层：Open WebUI

定位：

- 浏览器主入口
- 面向人交互，不承载业务逻辑

职责：

- 提供单一会话界面
- 上传题目附件、源码、日志、二进制
- 展示逻辑模型别名 `awdp-r1-70b`
- 保存历史会话和上下文
- 将所有请求发送给 `Agent Gateway`

输入：

- 用户消息
- 文件上传
- 会话历史

输出：

- 标准 OpenAI-compatible 请求
- 对话渲染结果

失败边界：

- 不参与模式判定
- 不参与工具路由
- 不直接连接 `llama-server`
- 不直接发现 MCP 工具

### 3.2 编排层：Agent Gateway / Orchestrator

定位：

- 系统控制面
- 单一策略入口

职责：

- 解析会话当前启用的模式组合
- 永远加载 `awdp` 主模式
- 按需叠加 `web` / `pwn` 插件提示词与约束
- 调用本地检索层获取上下文片段
- 构造发给模型的完整 prompt
- 约束模型只能输出固定 JSON action envelope
- 校验模型是否请求调用允许的工具
- 调 `Tool Router`
- 收集工具结果并组织二次推理
- 输出最终结构化回答
- 记录会话日志、模式、失败原因

输入：

- 来自 Open WebUI 的用户消息
- 文件摘要
- 会话历史

输出：

- 发往 `llama-server` 的推理请求
- 发往 `Tool Router` 的工具执行请求
- 返回给 Open WebUI 的最终答复

失败边界：

- 工具失败时仍必须给出结构化响应
- 插件冲突时以 `awdp` 主模式为准
- 不直接执行系统命令

### 3.3 推理层：llama-server

定位：

- 唯一模型推理服务

职责：

- 加载 `DeepSeek-R1-Distill-Llama-70B-Q4_K_M.gguf`
- 提供 OpenAI-compatible API
- 完成纯推理任务

输入：

- 来自 Gateway 的推理请求

输出：

- 模型回答
- 结构化 action envelope

失败边界：

- 不做工具治理
- 不做检索
- 不做业务逻辑
- 不做模式判断

### 3.4 工具层：Tool Router

定位：

- MCP 统一治理层

职责：

- 从 `D:\AI\Local\MCP` 读取工具注册表
- 按 `awdp` 主模式和 `web/pwn` 插件组合决定可用工具
- 校验工具名、参数、路径、超时
- 调用 MCP 服务
- 对结果做截断、摘要、错误归一
- 记录工具日志

输入：

- 来自 Gateway 的工具调用请求

输出：

- 工具执行结果
- 结构化错误
- 摘要化结果

失败边界：

- 任何工具越权直接阻断
- 工具不可用时不自动无限重试
- 不向 UI 暴露原始 MCP 细节

### 3.5 工具服务层：MCP Servers

定位：

- 功能执行单元

V1 固定接入：

- `filesystem/search`
- `terminal`
- `retrieval`
- `browser`
- `radare2`
- 可选 `docker`

职责：

- 按 MCP 协议提供具体能力
- 接受 Tool Router 的调用

失败边界：

- 每个服务独立超时
- 每个服务有独立工作目录限制
- 每个服务只处理自身职责，不跨层路由

---

## 4. 请求数据流

### 4.1 主数据流

固定调用链如下：

1. 用户在 Open WebUI 发起请求
2. Open WebUI 将请求发送到 `Agent Gateway`
3. Gateway 解析当前模式组合并构造 prompt
4. Gateway 调用检索层拿到知识片段
5. Gateway 将完整上下文发送给 `llama-server`
6. 如果模型返回 `tool_call`，Gateway 校验动作
7. 校验通过后，Gateway 调用 `Tool Router`
8. Tool Router 选择对应 MCP server 并执行
9. MCP 结果返回给 Tool Router
10. Tool Router 进行摘要和规范化
11. Gateway 将工具结果和中间状态发给 `llama-server` 进行二次推理
12. Gateway 输出最终结果给 Open WebUI

### 4.2 固定监听地址

- `llama-server`：`127.0.0.1:8080`
- `Agent Gateway`：`127.0.0.1:8090`
- `Tool Router`：`127.0.0.1:8091`
- `Open WebUI`：`127.0.0.1:3000`

### 4.3 Open WebUI 接入方式

- Open WebUI 只接 `Agent Gateway`
- 在 WebUI 内只暴露一个逻辑模型名：`awdp-r1-70b`
- WebUI 不直接感知：
  - 底层 GGUF 文件路径
  - MCP 服务器列表
  - 插件组合逻辑

### 4.4 工具调用协议

V1 固定不依赖模型原生 function calling 做主控制，改用 `JSON action envelope`。

标准结构：

```json
{
  "type": "answer | tool_call",
  "tool": "tool_name_or_empty",
  "arguments": {},
  "reason": "why_this_action"
}
```

规则：

- `type=answer` 时不触发工具调用
- `type=tool_call` 时必须校验：
  - `tool` 是否在白名单中
  - `arguments` 是否满足 schema
  - 访问路径是否在允许范围内
- 任何越权全部 `fail-closed`

---

## 5. 模式与插件加载机制

### 5.1 主模式

主模式固定为 `awdp`，永远加载：

- 系统提示词
- 输出格式
- 风险偏好
- patch 优先级
- 回归检查模板
- 公共工具白名单

### 5.2 子插件

V1 子插件固定为：

- `web`
- `pwn`

### 5.3 激活组合

允许的组合：

- `awdp`
- `awdp+web`
- `awdp+pwn`
- `awdp+web+pwn`

### 5.4 插件加载内容

每个插件只提供增量能力：

- prompt 片段
- skills 片段
- 工具白名单
- retrieval 源范围
- 评测标签

### 5.5 优先级规则

- `awdp` 永远最高
- `web` 和 `pwn` 只能补充，不能覆盖 `awdp`
- 复合题允许同时启用 `web` 与 `pwn`
- 插件冲突由 Gateway 按主模式规则裁决

---

## 6. MCP 接入与 Tool Router 设计

### 6.1 注册来源

所有 MCP server 定义集中放在：

- `D:\AI\Local\MCP`

Tool Router 启动时读取注册表，不从 UI 动态发现。

### 6.2 注册表字段

每个 MCP server 至少包含：

- `name`
- `command`
- `workdir`
- `timeout_ms`
- `allowed_paths`
- `plugin_scope`
- `transport`

说明：

- `plugin_scope` 取值为：`awdp`、`web`、`pwn`、或多值组合
- `transport` V1 默认优先 `stdio`

### 6.3 路由规则

`awdp` 默认工具：

- `filesystem/search`
- `terminal`
- `retrieval`

`web` 插件追加：

- `browser`
- `terminal` 的 web 命令模板

`pwn` 插件追加：

- `radare2`
- `terminal` 的 `checksec/pwntools` 命令模板

### 6.4 治理规则

Tool Router 必须记录：

- 会话 ID
- 当前模式组合
- 工具名
- 入参
- 执行时长
- 返回摘要
- 失败原因

### 6.5 失败策略

- 工具超时：返回结构化错误，不自动无限重试
- 权限拒绝：直接阻断并记录
- 服务不可用：Gateway 回退为无工具推理，并明确说明当前工具不可用
- 结果过大：Tool Router 先摘要，再回传

---

## 7. 本地知识库与检索方案

### 7.1 基本约束

V1 固定为不引入第二个本地 embedding 模型。

原因：

- 保持单主模型约束
- 降低本机常驻资源占用
- 避免引入第二套推理与维护路径

### 7.2 检索方案

检索采用三层：

1. 本地文件目录
2. `SQLite FTS5` 或等价轻量全文索引
3. `ripgrep` 兜底检索

知识片段由 Gateway 进行组装，再注入 prompt。

### 7.3 知识源分层

- `Core/AWDP`
- `Plugins/web/datasets`
- `Plugins/pwn/datasets`
- `Eval`
- `Writeups`
- `Templates`

### 7.4 V1 不做的内容

- 本地向量数据库
- 第二个常驻 embedding 模型
- 自动联网检索

---

## 8. 日志、审计与评测

### 8.1 会话日志

必须记录：

- 用户问题
- 当前模式组合
- 检索片段 ID
- 最终回答摘要

### 8.2 工具日志

必须记录：

- 工具名
- 参数
- 执行时长
- 成功 / 失败
- 结果摘要

### 8.3 评测日志

必须记录：

- 题目 ID
- 正确攻击面
- 是否调用对工具
- 是否给出可执行 patch / exp
- 是否误判
- 总耗时

### 8.4 安全边界

- Tool Router 只允许访问受控目录
- `terminal` 只暴露命令模板，不直接放开任意 shell
- `browser` 只访问本地靶场白名单地址
- `filesystem` 只开放题目目录、知识库目录、日志目录

---

## 9. 部署拓扑与端口规划

### 9.1 Windows 原生组件

- `llama-server`
- GGUF 模型文件
- 受控日志目录

### 9.2 WSL2 组件

- `Open WebUI`
- `Agent Gateway`
- `Tool Router`
- MCP Servers
- 检索索引与轻量数据库

### 9.3 网络与监听

- `Open WebUI`：`127.0.0.1:3000`
- `Agent Gateway`：`127.0.0.1:8090`
- `Tool Router`：`127.0.0.1:8091`
- `llama-server`：`127.0.0.1:8080`

要求：

- 所有服务默认仅监听本地回环地址
- V1 不做局域网暴露
- 不做公网访问

---

## 10. V1 实施顺序

### Phase 0：基础环境

- 安装 `llama.cpp / llama-server`
- 安装 `Open WebUI`
- 安装 `WSL2`
- 准备 `MCP` 运行环境
- 下载主模型 `DeepSeek-R1-Distill-Llama-70B-Q4_K_M.gguf`

### Phase 1：推理链路

- 启动 `llama-server`
- 用最小请求验证 OpenAI-compatible 接口
- 启动 `Agent Gateway`
- 让 Open WebUI 通过 Gateway 访问模型

### Phase 2：工具链路

- 启动 `Tool Router`
- 注册 `filesystem/search`
- 注册 `terminal`
- 注册 `retrieval`
- 再追加 `browser`
- 再追加 `radare2`

### Phase 3：模式与插件

- 固定 `awdp` 主模式
- 接入 `web` 插件
- 接入 `pwn` 插件
- 验证复合题模式加载

### Phase 4：日志与评测

- 打通会话日志
- 打通工具日志
- 打通评测日志
- 建首批回归用例

---

## 11. 附录：云训 / 外部训练链路

本机架构优先，本节只说明边界，不进入主数据流。

外部训练链路建议：

- 在云端或租卡环境做数据清洗、LoRA、蒸馏
- 输出新的训练样本、prompt 模板、插件数据、评测集
- 产物回灌到本机的：
  - `Core/AWDP`
  - `Plugins/web`
  - `Plugins/pwn`
  - `Eval`

不做的事：

- 不让本机依赖远端训练服务在线运行
- 不让本机实时联网拉取训练结果
- 不在创13本机执行 `70B` 微调

---

## 12. 验收场景

### 12.1 启动验证

- `llama-server` 单独可用
- `Agent Gateway` 可作为 OpenAI-compatible provider 被 Open WebUI 接入
- `Tool Router` 能正常注册并发现本地 MCP 服务

### 12.2 模式验证

- 纯 `awdp` 会话只加载核心 prompt 和核心工具
- `awdp+web` 会话只多出 web 插件内容和 web 工具
- `awdp+pwn` 会话只多出 pwn 插件内容和 pwn 工具
- `awdp+web+pwn` 会话可同时调用两类工具，且主规则仍来自 `awdp`

### 12.3 工具验证

- 文件检索工具能读取知识库
- `radare2` 工具能分析样例二进制
- `browser` 工具能访问本地靶场
- 非白名单工具调用会被阻断并记录

### 12.4 回归验证

- Open WebUI 端只配置一个逻辑模型入口
- Gateway 不因插件叠加破坏核心 prompt
- 工具失败时系统仍能给出结构化答复
- 日志中能完整还原一次会话的模式、工具、结果

---

## 13. 默认实现假设

- 推理后端固定为 `llama.cpp / llama-server`，不是 `vLLM`
- 主模型固定为 `DeepSeek-R1-Distill-Llama-70B-Q4_K_M.gguf`
- Open WebUI 是唯一主界面
- Open WebUI 只连接 `Agent Gateway`
- MCP 不直接散接到 Open WebUI，而是统一通过 `Tool Router`
- 本机架构优先，云训和外部训练只做附录
- V1 不引入第二个常驻模型，也不引入本地向量库
