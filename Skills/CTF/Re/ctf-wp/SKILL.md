---
name: ctf-wp
description: 面向本地 reverse 题解后的 WP 产出 skill。目标是低 AI 味、证据驱动、可复现，适合把 solved case 整理成正式或半正式 writeup。
---

# CTF Reverse WP

## 何时触发

在这些场景触发：

- 题目已经做出，用户要求“写 WP / 整理 WP / 给我一版 writeup”
- 题目已有 solve trace、review 结果、脚本或关键证据，需要整理成可看的文档
- 当前目标是“把做题过程沉淀成可复盘产物”，而不是继续盲目扩分析

如果题目还没有稳定结论，先回到 `rev-orch / rev-triage / reverse skills`，不要硬写成完整 WP。

## 目标

把题解整理成这条线：

`题型判断 -> 关键证据 -> 逻辑还原 -> 脚本验证 -> Flag`

WP 默认追求：

- 低 AI 味
- 不用第一人称
- 证据充足
- 步骤可复现
- 结构短而实

## 强制规则

1. 不使用第一人称  
   - 禁止：`我`、`我们`、`笔者`
   - 推荐：`通过`、`可见`、`该函数`、`最终可还原`

2. 不写 AI 套话  
   - 禁止：`本文将`、`接下来让我们`、`值得注意的是` 这类空转句

3. 不编过程  
   - 未直接验证的步骤必须标成“未直接验证”
   - 不伪造地址、工具输出、截图内容

4. 证据必须落地  
   - 至少包含一个硬证据点：
     - 字符串
     - 常量
     - 条件分支
     - 关键函数
     - 脚本输出

5. 默认相对路径  
   - 文中命令、脚本、分析文件优先使用相对路径
   - 不在 WP 正文里大量写绝对 Windows 路径

## 推荐结构

默认结构：

- `# Re - <ChallengeName>`
- `## 解题过程`
- `### 样本识别`
- `### 入口定位`
- `### 逻辑还原`
- `### 脚本验证`
- `## Flag`

如果题特别简单，可以压缩结构，但不能只剩一句结论。

## 写作检查表

写之前先确认：

- 最终 flag 是否已明确
- 关键证据是否至少有 2 个
- 是否已有可运行脚本或最小复现命令
- 是否需要把 partial solve 标清楚

写完后再确认：

- 全文没有第一人称
- 没有 AI 腔开场白
- 每节都有证据，不是纯感想
- 结尾有单独的 `## Flag`

## 产物建议

默认 challenge root 内建议这些产物：

- `WP/<challenge>_noai.md`
- `scripts/solve_<challenge>.py`
- `analysis/` 下保留必要中间证据

如果当前只是训练闭环样本，也可以先落在：

- `review_queue/<case_id>/wp-noai.md`
- `review_queue/<case_id>/wp-context.json`

## 参考

写之前可参考：

- `references/noai-style-checklist.md`
- `references/wp-template.md`
