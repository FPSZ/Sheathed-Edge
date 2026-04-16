# NSSCTF checkin AI WP

题目路径：
- `D:\CTF\题目\逆向\NSSCTF\已解决\checkin\checkin`

题型判断：
- `task_family: pwn/reverse-static`
- 字符串直埋 + 直接比较题

关键证据：
- 静态字符串里直接存在完整目标：
  - `moectf{Enjoy_yourself_in_Reverse_Engineering!!!}`
- 同时存在成功/失败提示
- 这类题通常是直接 `strcmp` 或轻量包装后比较

正确思路：
1. 先扫字符串。
2. 看到完整 flag 形态字符串后，不要继续过度深挖。
3. 只需验证它确实是比较目标即可。

易错点：
- 容易因为想“正规逆向”而错过最短路径
- 容易在简单题上浪费过多工具调用

人工结论：
- `moectf{Enjoy_yourself_in_Reverse_Engineering!!!}`
