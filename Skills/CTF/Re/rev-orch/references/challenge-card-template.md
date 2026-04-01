# Challenge Card Template

```yaml
challenge_root: ""
input_files:
  - path: ""
    role: archive|binary|attachment
file_type: ""
arch_platform: ""
suspected_family: ""
signal_strings: []
next_step: ""
```

字段要求：

- `file_type` 写题目主入口类型，不写模糊描述
- `suspected_family` 只写当前最可能的题型，例如 `base64-like`、`single-pe-check`、`upx-packed`
- `next_step` 必须是可执行动作

