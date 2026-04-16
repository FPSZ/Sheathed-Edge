# Review Checklist

- 是否先识别出这是“事务/状态机型二进制服务”
- 是否先恢复 `SET/GET/CLONE/MULTI/EXEC/ABORT` 的行为关系
- 是否明确 `TX START / TX EXEC / TX ABORT` 说明事务是独立层
- 是否把重点放在生命周期/状态转移，而不是急着套堆利用模板
- 是否能解释“重复 SET / 事务缓存 / refcount / tx chunk”之间的关系
- 是否把本题标记为需要函数级进一步核对后再升格 full teacher trace

