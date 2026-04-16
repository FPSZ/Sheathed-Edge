# NSSCTF attachment-15 AI WP

题目路径：
- `D:/CTF/题目/逆向/NSSCTF/已解决/attachment-15`

题型判断：
- `task_family: reverse`
- 这是典型的壳/压缩壳入门题，先识别 UPX，再决定是否需要脱壳。

关键证据：
- strings 中直接出现 `wctf2020{Just_upx_-d}`。
- 同时还能看到大量 ELF 运行时字符串，说明样本并不需要复杂算法恢复。
- 从 flag 内容本身也能反推题目核心点是 `upx -d`。

正确思路：
1. 先做文件 triage，判断是 ELF 可执行文件。
2. 看 strings / 壳特征，确认题目在考 UPX 脱壳意识。
3. 如果 strings 已经给出完整 flag，就应当直接收束；如果没有，再进入 unpack 路线。

易错点：
- 一上来就全量分析函数，忽略壳题先识别壳。
- 已有完整 flag 还继续深挖无关逻辑。

人工结论：
- `wctf2020{Just_upx_-d}`
