# Local Diff Check

- 本地二进制字符串确认存在 `MD5`、`seccomp_rule_add execve`、`ADD_SUCCESS/DEL_SUCCESS/EDIT_SUCCESS`、`RTSP server listening on 127.0.0.1:%d`。
- 这些特征与公开复盘中的『服务型 pwn + MD5 + 菜单操作 + seccomp 限制』高度一致。
- 结论：可视为高置信 candidate case，后续适合升级为 red/blue 双案。
