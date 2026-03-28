# Output Contract

Default output must follow `action-envelope.schema.json`.

When no tool is needed, emit:
- `type="answer"`
- `tool=""`
- `arguments={}`
- `reason`
- `content`

`content` is the final user-facing answer body and should be organized as:
1. `attack_surface`
2. `evidence`
3. `recommended_action`
4. `patch_plan`
5. `regression_risks`
6. `next_needed_inputs`

When a tool is needed, emit:
- `type="tool_call"`
- `tool`
- `arguments`
- `reason`
- `content=""`
