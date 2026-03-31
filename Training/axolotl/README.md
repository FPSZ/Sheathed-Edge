# Axolotl Reverse Training Scaffold

## 目标

这套脚手架用于把 `Datasets/reverse` 中已经过人工审查的样本转换成 Axolotl 可消费的 `chat_template` 训练数据，并生成可直接试跑的配置文件。

固定原则：

- 只读取清洗后的 `Datasets/reverse`
- 不直接读取 `Feed/Histroy`
- 不自动修改 `Eval/reverse`
- 不自动把未审查样本加入训练集
- 默认支持离线环境

## 目录

- `configs/base/`: 共享训练基线
- `configs/tasks/`: reverse 任务约束
- `configs/profiles/`: 运行档位
- `configs/generated/`: 生成后的最终配置
- `datasets/`: 训练转换输出
- `scripts/`: 校验、转换、配置生成脚本
- `outputs/`: Axolotl 训练输出目录

## 工作流

### 1. 校验 reverse 样本

```powershell
python Training/axolotl/scripts/validate_reverse_records.py
```

作用：

- 校验 `case/review/skill-delta/eval` 基本字段
- 拦截重复 `case_id`
- 拦截非法 split
- 拦截不允许进入 `eval` 的 case 被放进 `Eval/reverse/eval`

### 2. 生成 Axolotl 训练集

```powershell
python Training/axolotl/scripts/build_reverse_sft_dataset.py ^
  --allow-reviewed ^
  --output-dir Training/axolotl/datasets/reverse_pilot
```

默认行为：

- `train`: 来自 `Datasets/reverse` 中 `audit.can_use_for_train=true` 的 case
- `dev`: 来自 `task_meta.visibility=train-dev-only|dev-only` 的 case
- `eval`: 不生成训练 jsonl，只保留在 `Eval/reverse`

`--allow-reviewed` 用于第一阶段脚手架验证：

- 允许把 `reviewed but sft_ready=false` 的 case 转成训练样本
- 便于用当前已有的 `2026-newstar-yupi-adventure` 完成转换闭环

### 3. 生成 Axolotl 配置

```powershell
python Training/axolotl/scripts/render_axolotl_config.py ^
  --profile reverse-dry-run ^
  --train-file Training/axolotl/datasets/reverse_pilot/train.jsonl ^
  --val-file Training/axolotl/datasets/reverse_pilot/dev.jsonl ^
  --base-model Qwen/Qwen2.5-7B-Instruct ^
  --output-dir Training/axolotl/outputs/reverse-dry-run ^
  --save-path Training/axolotl/configs/generated/reverse-dry-run.yml
```

支持的 profile：

- `reverse-dry-run`
- `reverse-small-sft`
- `reverse-formal-sft`

### 4. 运行 Axolotl

预处理：

```powershell
python -m axolotl.cli.preprocess Training/axolotl/configs/generated/reverse-dry-run.yml
```

训练：

```powershell
accelerate launch -m axolotl.cli.train Training/axolotl/configs/generated/reverse-small-sft.yml
```

## 数据格式

转换后的 jsonl 使用 `chat_template` 兼容格式：

```json
{
  "messages": [
    {"role": "system", "content": "..."},
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "metadata": {
    "case_id": "...",
    "source_split": "train",
    "review_status": "reviewed",
    "skill_context": ["reverse-shell-adapter"]
  }
}
```

## 评测执行说明

评测不走 Axolotl 内置验证，继续使用仓库现有 `Eval/reverse`：

1. 选定 dev 题集
2. 让当前模型或新模型跑题
3. 按 `ReverseEvalRecord` 记录结果
4. 比较以下指标：
   - `solve_rate`
   - `time_to_first_useful_action`
   - `invalid_tool_call_rate`
   - `shell_mismatch_rate`
   - `error_recovery_turns`
   - `final_evidence_quality`

## 人工审查流程

进入训练前，样本必须满足：

- `case.json` 存在
- `review.json` 存在
- `skill-delta.json` 存在
- `audit.can_use_for_train=true`

推荐审查顺序：

1. 核对题目来源和可见性
2. 核对 student/teacher 差异
3. 写清 `error_tags`
4. 写清 `skill_delta`
5. 再决定是否把 `sft_ready` 置为 `true`
