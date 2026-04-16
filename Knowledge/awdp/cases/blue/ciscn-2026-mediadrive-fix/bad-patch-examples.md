# Bad Patch Examples

- 只新增 `preg_match` 关键字，不改 sink 前后的转换顺序
- 直接把 `encoding` 固定死但保留反序列化对象入口
- 直接返回 403 禁掉全部预览
- 只修 `preview.php`，完全不核对 `download.php` 语义是否一致

