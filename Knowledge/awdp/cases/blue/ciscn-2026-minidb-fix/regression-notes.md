# Regression Notes

- 需要回归：
  - `SET/GET/CLONE` 正常
  - `MULTI -> EXEC/ABORT` 正常
  - 重复事务操作不再触发 stale pointer/UAF
  - 服务无异常退出
