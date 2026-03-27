# Startup Order

在模型尚未下载完成前，可以先完成配置和目录整理。

## 推荐启动顺序

1. 准备主模型文件
2. 启动 `llama-server`
3. 启动 `Tool Router`
4. 启动 MCP servers
5. 启动 `Agent Gateway`
6. 启动 `Open WebUI`

## 上线前检查

- 主模型路径正确
- `gateway.config.json` 与 `tool-router.config.json` 端口不冲突
- `registry.json` 中的工具白名单与插件范围一致
- `Core/awdp`、`Plugins/web`、`Plugins/pwn` 三层配置存在
- 日志目录可写

