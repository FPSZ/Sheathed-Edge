# NSSCTF re1 AI WP

题目路径：
- `D:/CTF/题目/逆向/NSSCTF/已解决/re1.exe`

题型判断：
- `task_family: reverse`
- 这是标准的静态取证题，核心不是复杂算法，而是先确认程序里有没有直接可见的 flag 证据。

关键证据：
- 直接在字符串区能看到 `afctf{w41c0me a1l y0u 9uys}`。
- 入口逻辑没有比“先取字符串证据”更高优先级的复杂校验路径。

正确思路：
1. 先做最小 triage，确认它是普通 PE 可执行文件。
2. 优先看 strings，而不是先上复杂调试。
3. 当 strings 里已经出现完整 flag 形态时，应该直接收束，不要继续枚举函数浪费轮次。

易错点：
- 把这类送分题也当成复杂算法恢复题，导致白白消耗工具回合。
- 已经拿到完整 flag 还继续 list_functions / xrefs。

人工结论：
- `afctf{w41c0me a1l y0u 9uys}`
