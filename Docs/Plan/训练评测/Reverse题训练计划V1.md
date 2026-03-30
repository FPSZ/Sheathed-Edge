# Reverse 题训练计划 V1（Skills 优先，过程监督试点）

## Summary

这份计划以当前逆向题轨迹为起点，先做一条 `reverse` 试点闭环，再复制到其他题型。当前起始样本固定为：

- `Feed/Histroy/chat-export-1774864835443.json`
- `Feed/Histroy/26-3-30/chat-你好.txt`

核心决策固定如下：

- 第一阶段不直接上微调，先做 `skills + eval + 轨迹纠错`
- `reverse` 先作为试点，不一开始铺满 `awdp/web/pwn` 全域
- 训练重点不是只追最终做对，而是纠正过程质量：工具选择、shell 适配、路径编码、失败恢复、证据整理
- 第一阶段沿用现有 `awdp + pwn` 能力面，不先新建独立 `reverse` 插件
- 当前这道 `[re] 尤皮·埃克斯历险记` 已被看过，只能进 `train/dev`，不能进 `eval`

## Key Changes

### 1. 建立“题目-轨迹-纠错”数据闭环

原始聊天记录继续保留在 `Feed/Histroy`，不直接拿脏轨迹做训练；新增一层人工整理后的 `reverse case` 语料，最少包含这些字段：

- `case_id`
- `task_meta`
- `input_files`
- `student_trace`
- `teacher_trace`
- `error_tags`
- `gold_outcome`
- `skill_delta`
- `sft_ready`

每道题固定走这条流程：

1. 让当前模型独立做题并完整记录工具轨迹
2. 让更强 `teacher` 同题独立求解
3. 人工做差异审查，输出“错在哪、为什么错、理想过程是什么”
4. 把产物拆成三类：
   - `skill` 修订项
   - 过程监督样本
   - `eval` 候选样本

首批错误标签先从这次样本里抽象，不做泛化空话：

- `shell_mismatch`
- `path_or_encoding_failure`
- `missing_dependency_fallback`
- `powershell_syntax_error`
- `low_signal_exploration`
- `bad_error_recovery`
- `wrong_tool_priority`
- `poor_evidence_summary`

### 2. 先写小而可组合的 reverse skills，不写一坨大 prompt

第一阶段优先补 `skills`，因为当前问题更像“不会稳定按流程做”，不是“完全没有知识”。`skills` 固定拆成小块：

- `reverse-intake-and-path-normalization`
  - 先确认题目文件、压缩包、输出目录、编码与路径可达性
- `reverse-shell-adapter`
  - 明确当前是 `powershell / wsl-bash`，同一命令按宿主适配，不混 Linux/Windows 语法
- `reverse-archive-and-binary-triage`
  - 压缩包解包、文件类型识别、入口文件定位、基础字符串与元信息检查
- `reverse-static-analysis-escalation`
  - 何时先 `strings`，何时进 `radare2`，何时切动态验证
- `reverse-hypothesis-loop`
  - 每轮只提出少量假设，要求工具输出必须能支持下一步决策
- `reverse-final-report`
  - 证据、结论、未证实假设、复现步骤分开写

每个 skill 的接口固定包含：

- 触发描述
- 所需输入
- 编号流程
- 输出格式
- 最终检查项
- 必要时附脚本或参考文件

### 3. 训练重心放到“过程监督”，不是只喂最终题解

训练样本不要只保留最终答案或 `writeup`，必须保留可监督的中间决策。V1 不直接回灌原始自由链路思维文本，而是固定改成可控的结构化过程标签：

- `phase`
- `goal`
- `chosen_tool`
- `why_this_tool`
- `expected_evidence`
- `observed_result`
- `fallback_if_fail`

目标是把“正确过程”显式教给模型，尤其针对这次样本里暴露的多步失误。进入 `SFT` 之前，先累积 `teacher` 修正后的多轮过程样本，而不是只堆最终 `flag` 或最终解释。

### 4. 先做 reverse 评测集，再决定是否开训

第一阶段固定先做一个小而硬的 `reverse pilot` 数据集，建议 20 题起步：

- `train/curation`: 10
- `dev`: 5
- `eval`: 5

评测不只看“最后过没过”，还要固定记录这些过程指标：

- `solve_rate`
- `time_to_first_useful_action`
- `invalid_tool_call_rate`
- `shell_mismatch_rate`
- `avg_tool_calls`
- `error_recovery_turns`
- `truncated_output_incidents`
- `final_evidence_quality`

判定是否进入下一阶段的门槛固定为：

- `reverse dev/eval` 已稳定可复跑
- 至少积累 30 个人工校正过的 `reverse case`
- 经过 `skill` 与工具描述迭代后，主要失败模式已收敛，不再每题都换一种错法

### 5. 微调只作为第二阶段，不和第一阶段混做

达到上面门槛后，再准备外部训练。默认只做 `SFT`，不在 V1 上做 `DPO/RL`。

`SFT` 数据规则固定为：

- 保留多轮对话与工具调用结构
- 每条样本带上当前最优系统提示与 `skill` 上下文
- 训练集和测试集严格分离
- 高质量少量样本优先于大量低质量样本
- 先训“过程稳定性、工具调用偏好、错误恢复”，不是先训“写得像不像高手”

`DPO` 只留作后续可选项：

- 适合做“两个过程都能解，但哪个更稳更省工具”的偏好排序
- 不作为第一阶段主路线

## Artifacts / Interfaces

本计划新增的稳定接口不是 HTTP API，而是训练工件接口：

- `ReverseCaseRecord`
  - 一题的完整训练单元
- `TraceReviewRecord`
  - `student` 与 `teacher` 的差异审查结果
- `SkillDeltaRecord`
  - 某次失败应转化成哪条 `skill` 修订
- `ReverseEvalRecord`
  - 题目、预算、结果、工具指标、人工评分

建议后续落盘位置固定为：

- 原始轨迹：`Feed/Histroy`
- 清洗样本：`Datasets/reverse`
- 评测集：`Eval/reverse`
- `skill` 文档：沿现有 `Core/awdp/skills` 与 `Plugins/pwn/skills` 分层放置

## Test Plan

- 用当前这次样本先做 1 个完整示范 `case`，确认从原始轨迹到 `ReverseCaseRecord` 的整理规则可执行
- 再选 5 道本地 `reverse` 题，验证同一套错误标签是否够用，必要时只增补少量标签
- `skill` 初版完成后，重跑同一批 `dev` 题，确认这些指标确实改善：
  - 错 `shell` 命令显著下降
  - 路径/编码相关失败显著下降
  - 出错后能在 1-2 轮内切到合理 `fallback`
  - 工具调用更少但更有效
- 若 `skill` 迭代已明显提高 `solve_rate` 和过程指标，则继续扩样；若仍无明显改善，再进入 `SFT` 准备

## Assumptions

- 这份文档先作为 `Docs/Plan` 下的新训练计划
- 当前样本主要暴露的是流程性缺陷，不是单纯知识缺口，因此默认 `skills-first`
- 试点范围固定为 `reverse`，后续再复制到 `web/pwn`
- `reverse` 先挂在现有 `pwn` 能力层，不在第一阶段新开插件
- 训练与评测都要支持离线环境；比赛态不能依赖联网检索
- 评测集绝不回灌训练

## 参考依据

- OpenAI Skills：适合“可重复、多步骤、需要固定格式和检查项”的任务，且推荐拆成小的可组合工作流  
  <https://academy.openai.com/public/resources/skills>
- Anthropic Agent Skills：建议“先评测再补 skill”，并根据真实失败轨迹增量迭代  
  <https://claude.com/blog/equipping-agents-for-the-real-world-with-agent-skills>
- Anthropic Tools for Agents：工具改进应走评测驱动，重点看工具错误、冗余调用、描述和返回值是否高信号  
  <https://www.anthropic.com/engineering/writing-tools-for-agents>
- OpenAI Evaluation Best Practices：先做贴近真实分布的 held-out eval，并持续从日志挖边界 case  
  <https://platform.openai.com/docs/guides/evaluation-best-practices>
- OpenAI Fine-tuning Best Practices：高质量数据优先，训练/测试要先分开，并把最优提示上下文保留进训练样本  
  <https://platform.openai.com/docs/guides/fine-tuning-best-practices>
- OpenAI Process Supervision：对多步推理任务，监督过程通常比只监督结果更可靠  
  <https://openai.com/research/improving-mathematical-reasoning-with-process-supervision>
- NIST CAISI Cyber Evals：可借鉴“任务集 + agent + 预算 + 工具指标”的安全评测组织方式  
  <https://github.com/usnistgov/caisi-cyber-evals>
