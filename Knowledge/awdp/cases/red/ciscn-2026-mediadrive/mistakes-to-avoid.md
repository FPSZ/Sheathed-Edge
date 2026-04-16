# Mistakes To Avoid

- 不要一看到 `flag|/flag|..` 黑名单就草率判断“没戏”
- 不要把这题简化成普通 LFI；真正关键是“校验对象”和“消费对象”不是同一个值
- 不要只背 `ISO-2022-CN-EXT + fl%80ag`，那只是公开样例，不是技能本体
- 不要忽略 `basePath` 字段，它说明对象状态本身就是攻击面
- 不要只验证 `preview.php`，应顺手核对 `download.php` 是否更安全，帮助模型学会比较两条链

