# Teacher Notes

- Alexander17 的复盘给出的核心链是：`strcmp(MD5_raw, digest)` 鉴权绕过 -> `add(-1)` 导致 free 后 malloc 失败留下 UAF -> largebin attack 改记录大小 -> tcache poisoning -> 任意分配落到栈上劫持返回地址。
- 本地字符串中可见 `MD5`、`seccomp_rule_add execve`、`add/delete/edit/show`、`RTSP server listening`，与公开资料高度一致。
- Seyedog 的修复线索说得很清楚：把危险 UAF 处的 `call free` 换成已有 delete 流程，是可操作的最小补丁思路。
- 这题很适合做红蓝双视角 case：红队学服务型堆利用链，蓝队学 checker-safe 的 call target 替换修补。
