# Reverse Micro Skill: Packed / UPX

Use this when one or more of these are true:

- `main` is almost empty
- strings are thin or noisy
- a packer or UPX pattern is obvious
- the visible logic looks like a stub

Goal:

- avoid guessing from a hint string
- do one narrow packed-aware evidence step

Checklist:

- name the blocker precisely: UPX, packed stub, thin strings, or noisy strings
- if IDA strings are weak, do one wider raw strings pass on the original binary
- if a full flag already appears in raw strings, stop and answer
- if strings stay thin, state unpacking as the next narrow blocker

Hard rule:

- do not rewrite a hint string into a guessed flag format

Expected evidence:

- blocker summary
- wider strings result or unpack blocker

Fallback if fail:

- do not expand into a huge deobfuscation plan
- keep the next step narrow
