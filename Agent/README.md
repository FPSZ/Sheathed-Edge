# Agent 骨架

当前目录承载本地 AI 系统的控制面定义，不直接存放模型文件。

## 目录职责

- `gateway.config.json`
  - Agent Gateway 的固定配置
- `tool-router.config.json`
  - Tool Router 的固定配置
- `action-envelope.schema.json`
  - 模型输出动作协议
- `http-contract.md`
  - Open WebUI、Gateway、Tool Router、llama-server 之间的接口约定

## 设计原则

- Open WebUI 只接 Gateway
- Gateway 只接 `llama-server` 和 Tool Router
- Tool Router 只接 MCP Servers
- 所有模式与插件逻辑都在 Gateway 侧完成
- 所有工具权限与审计都在 Tool Router 侧完成

