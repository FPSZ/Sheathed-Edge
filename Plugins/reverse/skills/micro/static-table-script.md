# Reverse Micro Skill: Static Table Script

Use this when the blocker is exact table values, not control flow.

Common signals:

- static dword array
- static byte blob
- xor table
- target buffer
- compare buffer

Goal:

- read the exact data
- close the solve with one tiny script

Checklist:

- if a symbol exists, read by symbol first
- if only an address exists:
  - index table -> `ida_read_dword_array`
  - byte blob / target buffer -> `ida_read_byte_array`
  - literal string -> `ida_read_string`
- once the exact data is in hand, stop broad RE narration
- immediately script the final inversion or decode

Expected evidence:

- exact table / blob
- one short script
- final candidate

Fallback if fail:

- re-check width
- re-check signed vs unsigned handling
- re-check endianness
