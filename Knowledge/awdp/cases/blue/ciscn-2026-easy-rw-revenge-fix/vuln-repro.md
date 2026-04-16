# Vulnerability Repro Notes

- 公开复盘把核心漏洞指向 UAF。
- Blue 线索明确提到：危险点是某处分支直接 `call free`，导致后续对象仍被逻辑使用。
- 这类题不能只图省事把所有 free 全 NOP 掉，否则极易把服务状态机修坏。
