# Local Diff Check

- 本地解压样本包含 `catchme` 与 `libc-2.27.so`。
- 静态字符串已确认菜单项 `adopt/release/inspect/engrave/leave/purge` 与类型 `fox/hawk/otter`。
- 公开复盘中的修复点 `free 后清空 heap[idx]` 与本地接口高度匹配。
- 结论：本地/公开信息一致度高，但还缺本地函数级精核，所以暂定 medium-confidence。
