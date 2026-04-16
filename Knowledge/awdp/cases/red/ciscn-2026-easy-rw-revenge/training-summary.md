# Training Summary

## Why this case matters
这题适合训练模型把“网络服务 + 堆原语 + seccomp 限制”连起来看。

## Generalizable lesson
当二进制同时出现：
- 认证/摘要逻辑
- 网络监听/协议交互
- 菜单式对象管理
- seccomp 对 execve 的限制

模型应该优先考虑：
- 服务型漏洞复现
- 稳定 leak / write primitive
- ORW 或读 flag 路线
而不是默认本地 shell。

## Suitable use
- red exploit planning
- blue checker-safe fix planning
- pwn route correction for small models

