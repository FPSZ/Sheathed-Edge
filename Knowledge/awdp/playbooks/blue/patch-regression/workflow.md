# Workflow

## Regression set
- 固定四类检查：首页/健康检查、正常业务流、原 exploit、关键依赖资源
- 如是二进制服务，再加启动、监听、交互、异常输入

## Observe
- 记录状态码、正文片段、响应时延、进程状态、日志摘要
- 有条件时比较修补前后关键差异，不只看“能不能打开页面”

## Rollout decision
- 只有在 exploit 失效 + 正常路径稳定 + 资源无异常时才算可上场
- 不满足就回到 hotfix 或 checker-safe fix 阶段，不要硬推

