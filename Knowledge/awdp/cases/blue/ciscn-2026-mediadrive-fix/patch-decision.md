# Patch Decision

## Preferred patch lane
- 去掉对 `user` cookie 的直接反序列化
- 对最终 `convertedPath` 做规范化与目录限制
- 统一 preview/download 的路径策略

## Why this is the right level
- 漏洞核心不在前端展示，而在“用户状态 + 路径转换 + 最终 sink”这一层
- 只补字符串黑名单没有用，因为问题在校验时机错误
- 直接砍掉 preview/download 会破坏业务与 checker，不适合 AWDP

