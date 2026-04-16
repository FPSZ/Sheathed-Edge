# Route Correction

## Wrong route
- 上来只盯着 `flag|/flag|..` 黑名单
- 直接断言“有黑名单所以不能读 flag”
- 把题做成普通 LFI，忽略 cookie 里的对象状态

## Why it is wrong
- 真正消费的不是 raw path，而是 `iconv` 之后的 converted path
- `encoding` 和 `basePath` 都来自用户可控对象
- 这题关键不是黑名单内容，而是“校验值”和“消费值”不一致

## Correct route
- 先识别可控状态：`unserialize(cookie)`
- 再找 sink：`file_get_contents($convertedPath)`
- 再核对转换顺序：黑名单在前，`iconv` 在后
- 最后才讨论具体可用编码与样例

## Short correction phrase for model tuning
- 不要停在黑名单表面；先确认最终被 sink 消费的值是什么。

