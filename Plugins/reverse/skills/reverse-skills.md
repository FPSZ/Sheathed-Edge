# Reverse Family Skills

This is the short main skill for `task_family=reverse`.

Use it after:

- `task_family: reverse`
- `shared_domain: binary`
- `binary-core` has already produced the first evidence set

Do not turn this file into a knowledge dump.

If a sub-pattern becomes obvious, load one matching micro skill from `skills/micro/` instead of expanding this file.

## Stable Fields

Keep these fields visible when possible:

- `task_family: reverse`
- `shared_domain: binary`
- `phase`
- `primary_skill`
- `evidence_required`
- `fallback_if_fail`

## Reverse Main Route

Default route:

1. open the target in the most stable binary tool
2. get one real metadata result
3. get one real strings/functions result
4. choose one concrete xref / caller / callee / decompile target
5. extract exact static data if the logic depends on tables, blobs, or target bytes
6. once the data is enough, run one tiny verification script
7. close with the exact flag / key if the remaining gap is only a narrow transform

Do not spend multiple turns narrating the plan after step 5.

## IDA Typed Tool Opening

Use this path when the local IDA bridge is available.

Checklist:

- call `ida_open_file`
- call `ida_get_metadata`
- call either `ida_list_strings` or `ida_list_functions`
- choose one concrete suspicious address / function
- call one of:
  - `ida_get_xrefs_to`
  - `ida_get_callers`
  - `ida_get_callees`
  - `ida_decompile_function`
- if the clue points to exact static data, immediately switch to:
  - `ida_get_global_variable_value_*`
  - `ida_read_dword_array`
  - `ida_read_byte_array`
  - `ida_read_string`

Expected evidence:

- current binary metadata
- one suspicious clue
- one next target

Fallback if fail:

- if `ida_open_file` fails, fall back to the next stable binary tool
- if the bridge is stale, try `ida_close_active_session` once, then reopen

## Key Function Recovery

Use this when the blocker is still “which function matters?”

Checklist:

- connect one string / import / branch to one function
- prefer one xref or caller step over broad traversal
- reduce the search space to one or two high-signal targets
- reject weak candidates explicitly

Expected evidence:

- key function candidate
- supporting clue
- narrow next target

Fallback if fail:

- if names are stripped, pivot to strings and xrefs
- if strings are weak, pivot to entry-adjacent logic or imports

## Check Tracing

Use this when the blocker is “what must be true for success?”

Checklist:

- decompile the check function
- identify compare points, branch conditions, or table lookups
- summarize the exact success condition
- name the exact missing evidence before extraction

Expected evidence:

- check routine summary
- decisive compare / branch / constant

Fallback if fail:

- if the trace is noisy, reduce scope to one branch or one compare site
- if the path is indirect, step back to callers

## Data Extraction

Use this when the blocker is no longer control flow, but exact values.

Checklist:

- if a symbol exists, prefer `ida_get_global_variable_value_*`
- if it is an index table, prefer `ida_read_dword_array`
- if it is a byte blob or target buffer, prefer `ida_read_byte_array`
- if it is a literal string, prefer `ida_read_string`
- only use raw memory reads when the higher-level read fails

Hard rule:

- once exact data is in hand, stop broad RE narration and move to one tiny script

Expected evidence:

- exact table / blob / target bytes

Fallback if fail:

- confirm width and endianness with a smaller read first

## Script Verification

Use this when one short script can finish the solve.

Checklist:

- use one single-purpose script only
- script types that should close quickly:
  - custom alphabet decode
  - xor / add / sub inversion
  - permutation inversion
  - index-based reorder
  - small arithmetic normalization
- print only decisive output

Hard rule:

- do not hand-calculate a transform once the exact data is already available

Expected evidence:

- candidate flag / key
- one short reproducible solve step

Fallback if fail:

- go back to operation order, index order, or one unresolved constant

## Exact Answer Closure

Use this when only one narrow step remains.

Checklist:

- cite the decisive function / table / transform clue
- if the gap is only `base64`, custom alphabet, xor, permutation, or one common mutation such as `o -> 0`, finish it
- do not stop at “best candidate” if one more reversible step gives the exact answer
- do not leave tool-planning text in the final answer

Expected evidence:

- final flag / key
- short extraction chain

Fallback if fail:

- return to the closest unresolved transform edge

## Packed / Thin Main Fallback

Use this when `main` is too empty or strings are too thin.

Checklist:

- name the blocker precisely: packed, UPX, stub main, junk flow, or stripped clue path
- if IDA strings are thin, do one wider raw strings pass on the original binary
- if a full flag already appears in strings, stop and answer
- do not turn this into a huge deobfuscation project unless the task truly requires it

Expected evidence:

- blocker summary
- next smallest workaround

## Companion File Rule

Use this when the binary obviously depends on a sibling artifact such as:

- `output.txt`
- `enc.txt`
- `input.txt`
- `result.txt`

Rules:

- inspect the companion artifact before inventing a static data source
- do not treat an arbitrary `.rdata` / `.rodata` blob as expected output if the program behavior points to a file
- once the file data is known, invert with one short script

## Micro Skill Dispatch

If the pattern is obvious, load one matching file from `skills/micro/`:

- `custom-alphabet.md`
- `static-table-script.md`
- `permutation-xor.md`
- `packed-upx.md`
- `companion-file.md`

Do not load multiple micro skills unless the evidence really spans multiple patterns.

## Writeup Rule

Only switch to writeup mode when the user asks for WP / writeup.

Then:

- keep it short
- keep it factual
- cite concrete evidence
- separate solved facts from any remaining uncertainty
