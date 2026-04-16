# Patch Decision

## Preferred patch lane
- 找到危险 `call free` 分支
- 改为调用已有 delete/cleanup 逻辑
- 保证对象指针与元数据同步更新

## Why this is the right level
- 公开复盘已经提示这类改法更接近 checker-safe 最小修补
- 大改对象生命周期风险太高
- 粗暴 NOP free 往往会造成服务行为漂移或资源状态异常

