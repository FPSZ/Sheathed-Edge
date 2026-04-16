# Route Correction

## Wrong route
- 一看到堆题就直接进入 one_gadget / house-of-xxx 话术
- 忽略它是服务型二进制
- 忽略 seccomp 对最终利用目标的限制

## Why it is wrong
- 这题不是本地单进程菜单题，而是服务侧交互
- 出现 `RTSP server listening` 和 `seccomp_rule_add execve`，说明利用目标选择会被环境约束
- 如果原语都没确认，就直接套 largebin/tcache，只会让小模型越来越飘

## Correct route
- 先识别服务属性与协议约束
- 再确认认证 / 参数异常 / UAF 原语
- 再决定是否进入 largebin / tcache 组合利用
- 最后根据 seccomp 选择 ORW / 读 flag 而非默认 shell

## Short correction phrase for model tuning
- 先证明 primitive，再选择 exploit family；服务约束会改变最终目标。

