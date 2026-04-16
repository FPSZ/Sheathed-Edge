# Workflow

## Harvest
- 先确认 flag 可能位置、格式、编码、行尾
- 如果回显噪声大，先提纯再提交
- 如果需要多步读取，保存最短链路，不混入多余探测

## Submit
- 统一记录目标、时间、flag 值、来源路径、提交结果
- 先人工确认一条成功提交样本，再自动化
- 对重复 flag、空 flag、旧轮 flag 做去重

## Post-submit
- 若 exploit 仍可复用，保留最小模板
- 若目标变更，重新回到 exploit reuse，而不是盲提重复值

