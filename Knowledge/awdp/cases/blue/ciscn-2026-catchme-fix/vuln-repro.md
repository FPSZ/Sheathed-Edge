# Vulnerability Repro Notes

- 公开复盘确认 `delete/release` 存在 UAF。
- 根因是 free 后对应记录位/槽位未清空，后续 inspect/engrave 仍可命中旧块。
