# BUUCTF level1 AI WP

题目路径：
- `D:\CTF\题目\逆向\BUUCTF\level1\level1`
- `D:\CTF\题目\逆向\BUUCTF\level1\output.txt`

题型判断：
- `task_family: pwn/reverse-static`
- 文件读取 + 简单算术变换逆推

关键证据：
- 主逻辑读取本地 `flag` 文件前 20 字节
- 从第 1 位到第 19 位循环输出数字
- 奇数索引：字符左移 `idx`
- 偶数索引：字符乘 `idx`
- `output.txt` 中给出了变换后的数字序列

正确思路：
1. 先识别程序不是直接校验输入，而是把真实 flag 变换后打印。
2. 从 `main` 反推出每一位的运算规则。
3. 用输出反推原始字符。

易错点：
- 容易忽略索引从 1 开始，不是从 0 开始
- 容易把所有位都按同一规则还原

人工结论：
- `ctf2020{d9-dE6-20c}`
