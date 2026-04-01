# PowerShell Reverse Cheatsheet

## 解压

- 首选：系统自带解压或 .NET `ZipFile`
- 不默认依赖 `7z`

## 列目录

- 优先 `Get-ChildItem`
- 需要递归时明确 `-Recurse`

## 文本搜索

- 优先 `rg`
- 缺失时退回 `Select-String`

## 字符串扫描

- 优先现成工具
- 缺失时只做有限、目标明确的文本扫描

## 原则

- 命令写清目标路径
- 尽量不要对整个磁盘或整个仓库扫
- 避免一次性输出巨量低信号内容

