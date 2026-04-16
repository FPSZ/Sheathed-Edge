# Checker-safe Notes

- 保持原协议与交互返回尽量不变
- 保留 add/delete/edit/show 正常语义
- 修补后要重点看：服务是否还正常监听、是否还能正常收发协议
- 如果单点 call target 替换就能封住 UAF，就不要继续扩修范围

