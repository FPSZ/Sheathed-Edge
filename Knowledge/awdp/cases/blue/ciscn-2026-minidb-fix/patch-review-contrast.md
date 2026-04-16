# Patch Review Contrast

## Good patch lane
- 锁定事务分支里的错误释放 / stale pointer 更新点
- 修事务生命周期，不粗暴破坏普通命令语义
- 明确区分 `SET/GET/CLONE` 与 `MULTI/EXEC/ABORT` 的修补边界

## Bad patch lane A: NOP all frees in transaction path
- 看起来能止血
- 但容易导致事务对象泄漏、状态错乱、服务异常

## Bad patch lane B: treat as generic heap bug only
- 只从 malloc/free 层面补
- 忽略这题本质上是 transaction-state bug

## Preferred judgement
- 这题蓝队训练重点是“选对补丁层级”：修状态机生命周期，而不是只修某个 free 指令。

