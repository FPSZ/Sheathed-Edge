# Patch Review Contrast

## Good patch lane
- 去掉用户可控反序列化
- 对最终转换后的路径做规范化与目录约束
- 统一 preview/download 策略

## Bad patch lane A: only add more blacklist words
- 看起来快
- 实际仍然可能被编码转换、规范化差异绕过

## Bad patch lane B: directly disable preview/download
- 看起来安全
- 实际会破坏业务与 checker

## Preferred judgement
- 这题应该修“边界语义”，不是继续修 payload 词表。

