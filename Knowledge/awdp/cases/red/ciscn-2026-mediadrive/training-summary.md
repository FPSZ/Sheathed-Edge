# Training Summary

## Why this case matters
这题适合训练小模型理解：
- 用户可控对象状态
- 编码转换前后值不一致
- 安全校验必须打在最终消费值上

## Generalizable lesson
如果系统存在：
- 用户可控 `encoding / charset / locale / path-base`
- 先做字符串黑名单，再做转换或规范化
- 最终 sink 消费的是转换后的值

那就应优先怀疑“前检查、后消费”的不一致漏洞，而不是停在原始字符串表面。

## Suitable use
- red route correction
- web sink judgement
- blue minimal hotfix design

