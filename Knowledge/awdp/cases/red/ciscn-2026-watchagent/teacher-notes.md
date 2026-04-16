# Teacher Notes

- 公开复盘把这题描述成一个简单的 agent/socket 服务，关键函数名类似 `Agent::handleClient / receiveCommand / executeCommand`，最终通过 `popen` 执行命令。
- 同一来源还提到了认证 token `RCE_AUTH_2026`，但在没有本地二次核对之前，不能把它直接当标准答案字段写死进训练集。
- 本地样本字符串能看到 `WatchAgent RPC Server`、`Command executed successfully. Exit code: %lu` 和 PDB 路径，说明它确实是命令代理型服务。
- 这题当前更适合作为渗透/ISW 弱监督案例，用于教模型『先识别服务接口与认证，再决定是 web 入口还是 agent 协议』。
