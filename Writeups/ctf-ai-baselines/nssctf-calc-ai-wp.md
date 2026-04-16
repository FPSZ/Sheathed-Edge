# NSSCTF calc AI WP

题目路径：
- `D:/CTF/题目/逆向/NSSCTF/已解决/calc`

题型判断：
- `task_family: reverse`
- 这是“strings 直接暴露答案”的入门题，考的是会不会先做低成本证据提取。

关键证据：
- 可打印字符串中直接出现 `utflag{str1ngs_1s_y0ur_fr13nd}`。
- flag 内容本身也在提示做题方法：先看 strings。

正确思路：
1. 先确认是 ELF/PE 普通二进制。
2. 第一轮直接 strings 扫描。
3. 发现完整 flag 后立刻给出答案，并说明“这是字符串直出题”。

易错点：
- 看见 `calc` 文件名就误判成复杂运算恢复题。
- 忽视 flag 文本本身给出的提示语义。

人工结论：
- `utflag{str1ngs_1s_y0ur_fr13nd}`
