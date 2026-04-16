# Patch Review Contrast

## Good patch lane
- 在释放后立即清空 `heap[idx]` 或对应记录位
- 只补 dangling reference 的断开逻辑
- 尽量不动其它菜单分支与类型选择逻辑

## Bad patch lane A: patch around exploit sequence only
- 只针对已知 exp 里的某几个 index 或尺寸做特判
- 这会导致补丁高度特化，换个利用序列就失效

## Bad patch lane B: disable inspect/engrave wholesale
- 看起来能阻断利用
- 但通常会破坏业务与 checker 路径

## Preferred judgement
- 这题正确修法应当是“释放后断引用”，而不是“按 exp 反推封洞”。

