# Workflow

## Model the checker surface
- 明确 URL、参数、Cookie、状态码、正文关键字、延迟容忍
- 不知道 checker 行为时，优先从现有正常请求与日志里反推

## Fix with compatibility
- 尽量保持原路由、原参数名、原成功码、原页面骨架
- 如果必须拒绝恶意输入，优先返回兼容的业务错误而不是异常堆栈
- 对高风险分支加鉴权、约束或白名单，而不是删整个功能

## Validate
- 先回放正常请求
- 再回放 exploit 请求
- 最后观察服务资源占用、日志异常、重启情况

