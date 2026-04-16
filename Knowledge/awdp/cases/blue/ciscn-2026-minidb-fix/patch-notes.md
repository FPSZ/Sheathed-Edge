# Patch Notes

- 最小修补原则：修事务分支中的错误释放/悬垂指针，不要把所有 free 粗暴 NOP。
- 当前推荐策略：先把这题列为“需要函数级核对后再固化补丁”的 blue candidate。
- 为什么不走大改：事务语义复杂，比赛中大改容易把 `MULTI/EXEC/ABORT` 整体修坏。
