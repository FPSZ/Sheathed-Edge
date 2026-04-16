# Local Diff Check

- 本地 `preview.php` 确认存在 `@unserialize($_COOKIE['user'])`。
- 本地 `lib/User.php` 确认对象字段为 `name / encoding / basePath`。
- 本地 `preview.php` 确认黑名单检查发生在 `iconv(...)` 之前，且真正读取的是 `$convertedPath`。
- 本地 `preview.php` / `download.php` 结构与公开复盘中提到的 `preview.php?f=`、调试显示 converted path 明显一致。
- 结论：本地样本与公开解法高度匹配，可作为高置信 candidate case。
