# Vulnerability Repro Notes

- 公开资料把问题集中在 `MULTI` 事务实现：重复 `SET`、旧 value 释放、refcount、tx chunk 生命周期。
- S1nyer 还提到过把危险 `call _free` 替换到未使用逻辑 `sub_13BE` 一类思路，但在本地函数级核完前不要把它当唯一答案。
