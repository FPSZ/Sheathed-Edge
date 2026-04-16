# NSSCTF 世界最高のプログラマーです AI WP

题目路径：
- `D:/CTF/题目/逆向/NSSCTF/已解决/世界最高のプログラマーです.exe`

题型判断：
- `task_family: reverse`
- 这是轻量逻辑校验题，但当前样本里 flag 其实已经被字符串直接泄露。

关键证据：
- strings 中出现：`Hello CTFer~!`
- 输入提示：`Input 2 numbers which are the C0RE of the computer`
- 直接出现完整答案：`Flag: LitCTF{I_am_the_best_programmer_ever}`

正确思路：
1. 先从 strings 判断程序交互形态。
2. 发现完整 flag 已经直出后，不再把它当复杂数值校验题深挖。
3. 解释这题更像“做最小 triage 就能收束”的样本。

易错点：
- 被“输入两个数字”这句误导，过早进入方程恢复。
- 忽略字符串区已经给出的最终答案。

人工结论：
- `LitCTF{I_am_the_best_programmer_ever}`
