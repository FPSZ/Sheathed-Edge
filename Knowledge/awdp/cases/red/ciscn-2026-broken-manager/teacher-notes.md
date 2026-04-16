# Teacher Notes

- 公开复盘把重点放在自实现小分配器，而不是 glibc 常规模板。
- 核心攻击关键词：double free / UAF、SIGSEGV 崩溃信息用于恢复 arena 地址、借 `sigaltstack` 让流程重新回到 `main`，最后 freelist poisoning 命中 altstack 返回地址。
- 这种 case 的价值不在某个 one-liner exp，而在于教模型先判断『题目是否使用自定义 allocator / 异常恢复逻辑』。
- 这题建议后续补一份更细的 teacher trace，再用于 pwn family 强监督。
