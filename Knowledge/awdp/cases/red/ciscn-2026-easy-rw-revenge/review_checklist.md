# Review Checklist

- 是否先判成“服务型 pwn”，而不是本地一次性交互题
- 是否明确识别 `MD5`、`add/delete/edit/show`、`RTSP server listening`、`seccomp`
- 是否先找稳定原语，再谈 largebin/tcache
- 是否说明 seccomp 限制会影响最终目标选择（更偏 ORW/读 flag）
- 是否能区分：鉴权问题、UAF 触发点、后续堆利用阶段
- 是否在 blue 视角中给出“替换危险 free 路径”的最小修补思路

