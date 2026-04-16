# Bad Patch Examples

- 把事务相关 free 全 NOP
- 不区分普通命令与事务命令的生命周期差异
- 只修单条命令崩溃，不验证 `MULTI/EXEC/ABORT` 整体语义
- 直接借历史同名题 patch 思路，不核本地结构

