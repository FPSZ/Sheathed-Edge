# Patch Notes

- 最小修补点：把危险 `call free` 改成调用现成 delete/清理逻辑，让指针与元数据一起更新。
- 为什么不走大改：比赛中最重要的是保留协议行为和 checker 路径；替换 call target 通常比重构对象生命周期更稳。
- 回滚点：只改单处分支跳转 / call 目标，配套记录原始 offset。
