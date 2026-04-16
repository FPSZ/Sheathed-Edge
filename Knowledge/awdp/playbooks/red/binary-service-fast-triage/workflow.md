# Workflow

## Intake
- 确认架构、位数、动态/静态链接、监听端口、交互方式、是否带 menu / auth / file op
- 抓一份最小正常交互 transcript
- 先看 strings / imports / symbols / usage 文本，找高价值功能点

## Fast judgement
- 快速判断保护：NX、PIE、Canary、RELRO、FORTIFY，外加协议长度控制
- 找高风险入口：读取长度、格式化输出、命令执行、文件路径、反序列化、自定义协议 parser
- 如果是明显逻辑洞或调试残留，优先走最短逻辑链，不盲目撞内存破坏

## Narrow verification
- 用最小输入验证：崩不崩、能不能泄漏、是否可控长度、是否有回显差异
- 记录每一步输入、响应、断连、重连行为
- 如果还没有原语，不扩大战线，先回到 strings / funcs / xrefs 或协议整理

## Handoff
- 形成 exploit 原语后，转到 exploit reuse / scaling
- 若价值不高或难度明显过高，及时止损，转向更易得分目标

