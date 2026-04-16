# Teacher Notes

- 公开复盘把主链指向 `preview.php?f=` 文件读取。
- `user` cookie 直接 `unserialize`，可控 `User` 对象中的 `encoding` 与 `basePath`。
- 本地 `preview.php` 先拼接 `$rawPath = $user->basePath . $f`，再对 rawPath 做 `flag|/flag|..|php:|data:|expect:` 黑名单。
- 黑名单发生在 `iconv($user->encoding, 'UTF-8//IGNORE', $rawPath)` 之前，因此可先构造不会命中 `/flag` 的原始字节串，再借编码转换落到 `/flag`。
- 公开资料给出的关键例子是把 cookie 中 `encoding` 设为 `ISO-2022-CN-EXT`，随后访问 `preview.php?f=fl%80ag`，页面调试区能看到 `Raw path=/fl�ag`、`Converted=/flag`。
- 红队训练时应先把这题沉淀成『可控编码转换绕过路径黑名单』案例，而不是只背某个 `%80`。
- 蓝队 fix 方向：禁止反序列化用户输入；路径校验必须落在转换后的规范化路径上；下载/预览统一限制到受控上传目录。
