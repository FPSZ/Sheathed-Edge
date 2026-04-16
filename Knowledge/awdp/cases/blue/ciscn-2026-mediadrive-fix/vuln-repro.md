# Vulnerability Repro Notes

- `preview.php` 先对 raw path 做黑名单，再对 raw path 进行 `iconv`，最后读 converted path。
- `user` cookie 可直接反序列化为 `User`，使 `encoding/basePath` 可控。
- 公开复盘给出通过编码转换把非 `/flag` 原始串变成 `/flag` 的利用。
