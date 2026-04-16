# Route Correction

## Wrong route
- 看到菜单堆题就直接套 house-of-storm 细节
- 不先确认菜单语义和 chunk 尺寸关系
- 只记公开 exp 里的命令序列，不理解为什么要这样布局

## Why it is wrong
- 这题真正有训练价值的是：
  - 多尺寸 chunk 题型识别
  - leak 阶段与利用阶段分离
  - AWDP 蓝队视角下的“释放后断引用”修法
- 如果只背 House of Storm 步骤，小模型换个尺寸或菜单名就会崩

## Correct route
- 先恢复菜单：`adopt/release/inspect/engrave/leave/purge`
- 再恢复三种对象尺寸：`fox / hawk / otter`
- 先判断哪条路径负责 leak libc，哪条路径负责后续布局
- 最后再进入 House of Storm 这类 exploit family

## Short correction phrase for model tuning
- 先恢复对象种类与尺寸，再讨论高级堆技法。

