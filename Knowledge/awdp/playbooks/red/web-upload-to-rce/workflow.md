# Workflow

## Intake
- 确认上传入口、认证要求、字段名、文件大小限制、扩展名限制、返回体结构
- 判断返回的是相对路径、URL、文件 ID，还是完全不回显路径
- 先上传无害小文件，确认落盘和访问路径

## Execution path judgement
- 若服务端可直接访问上传目录，优先测试脚本执行型后缀、双后缀、解析差异
- 若不可直达，测试 include / template import / plugin load / image parser / office parser / unzip 落点
- 若返回的是文件 ID 或下载接口，继续判断是否存在路径拼接、文件类型识别错误、预览器执行

## Verification
- 每次只改一个变量：文件名、后缀、Content-Type、边界、压缩包结构、目录层级
- 先验证一次最小命令执行或一次明确文件包含证据
- 形成成功样本后立即模板化

## Harvest / Reuse handoff
- 成功后马上记录：上传请求、访问路径、最小 payload、实例差异
- 不在同一轮里同时尝试复杂内存马、批量投递、持久化改造

