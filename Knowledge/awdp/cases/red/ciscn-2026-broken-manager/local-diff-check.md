# Local Diff Check

- 本地样本路径、题名与公开复盘一致。
- 对本地二进制做静态字符串检查时能看到 `sigaltstack` 与菜单项 `3. Show`，与公开复盘的关键字部分一致。
- 但当前还没有完成本地函数级核对（如 allocator 结构、异常路径、复入点），因此 `verified_local_match=false`。
- 结论：可以先纳入高价值 candidate，但训练时仍应要求本地二次核对。
