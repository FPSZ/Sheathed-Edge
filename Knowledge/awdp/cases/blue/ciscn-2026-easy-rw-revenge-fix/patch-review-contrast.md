# Patch Review Contrast

## Good patch lane
- 锁定危险 `call free` 分支
- 替换为已有安全 delete/cleanup 逻辑
- 尽量不动协议层和其它菜单行为

## Bad patch lane A: NOP every free-like site
- 看起来能快速止血
- 实际容易导致资源状态漂移、checker 崩溃、协议异常

## Bad patch lane B: full lifecycle rewrite during match
- 理论上更完整
- 但 AWDP 比赛窗口里风险太高，最容易引入新 bug

## Preferred judgement
- 这题优先训练“最小 call target 替换”，不是“赛中重构内存管理”。

