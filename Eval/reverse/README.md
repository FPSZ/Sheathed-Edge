# Reverse Eval Layout

`reverse` 评测和训练样本严格分开。

- `dev/`: 可反复调参的开发集
- `eval/`: 只用于阶段验收的保留集
- `templates/`: `ReverseEvalRecord` 模板

评测记录至少保留：

- `solve_rate`
- `time_to_first_useful_action`
- `invalid_tool_call_rate`
- `shell_mismatch_rate`
- `error_recovery_turns`
- `final_evidence_quality`
