# Reverse Micro Skill: Companion File

Use this when the binary obviously depends on a sibling artifact such as:

- `output.txt`
- `enc.txt`
- `input.txt`
- `result.txt`

Goal:

- take the real external artifact as evidence
- avoid treating unrelated static data as expected output

Checklist:

- confirm from decompile or behavior that the program reads, writes, or prints external file data
- inspect the companion file first
- extract the exact numbers / bytes / lines from that file
- write one short script to invert the transform

Hard rule:

- do not treat an arbitrary `.rdata` / `.rodata` blob as the expected output when a companion file exists

Expected evidence:

- companion file contents
- transform rule
- final candidate

Fallback if fail:

- re-check whether the program reads the file or writes the file
- re-check indexing origin and line order
