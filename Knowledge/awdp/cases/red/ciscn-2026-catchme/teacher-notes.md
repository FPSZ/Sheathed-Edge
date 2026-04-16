# Teacher Notes

- Seyedog 复盘把蓝队修复讲得最清楚：`delete/release` 存在 UAF，核心补丁是在 `free` 后把 `heap[idx]` 置空。
- S1nyer 复盘把攻击面展开成了 House of Storm 路线，并给出了菜单：`adopt / release / inspect / engrave / leave / purge`，生物类型是 `fox / hawk / otter`。
- 本地解压后的二进制字符串完全能对上这套菜单和类型名，因此可确认至少题型与接口是一致的。
- 训练上不应只记 one-gadget，而应记『多尺寸 chunk -> 先 leak libc -> 再构造 House of Storm』的阶段化思路。
