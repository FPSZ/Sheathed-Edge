# simplebase AI WP

题目路径：
- `D:\CTF\题目\逆向\NSSCTF\未解决\06base64-\simplebase`

题型判断：
- `task_family: pwn/reverse-static`
- 更准确说是轻量逆向中的编码还原题

关键证据：
- 主程序把输入经过自定义字符表做 base64 编码
- 自定义表为：
  - `NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm0123456789+/`
- 对照标准 base64 可判断字母区做了 ROT13 变换
- 密文字符串：
  - `GyAGD1ETr3AcGKNkZ19PLKAyAwEsAIELHx1nFSH2IwyGsD==`

正确思路：
1. 不要先暴力猜 flag。
2. 先识别它不是解码函数，而是编码函数。
3. 再把自定义表映射回标准 base64 表。
4. 将密文按映射还原，再做 base64 解码。

易错点：
- 容易卡在“手工逐字符推导”而不是直接建立映射表
- 容易把自定义表误认成简单替换而忽略它本质仍是 base64

人工结论：
- `NSSCTF{siMp13_Base64_5TXRMZHU6V9S}`
