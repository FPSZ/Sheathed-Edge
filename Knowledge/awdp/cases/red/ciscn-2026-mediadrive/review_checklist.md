# Review Checklist

- 是否先把题判成 `web`，而不是直接开始猜 payload
- 是否先读了 `preview.php / download.php / profile.php / lib/User.php`
- 是否明确指出 `user` cookie 被直接 `unserialize`
- 是否明确指出黑名单检查发生在 `iconv` 之前
- 是否明确指出真正 sink 是 `file_get_contents($convertedPath)`
- 是否把“raw path 与 converted path 不一致”作为主结论，而不是只记某个 `%80`
- 是否说明 `download.php` 与 `preview.php` 的约束差异
- 是否在给出 patch 建议时强调“校验最终规范化路径 + 去反序列化”

