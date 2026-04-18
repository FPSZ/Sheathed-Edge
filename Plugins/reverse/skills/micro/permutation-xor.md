# Reverse Micro Skill: Permutation + XOR

Use this when the challenge length is fixed and the transform looks like:

- reorder by index table
- then xor / add / sub by index

Goal:

- recover the forward order
- invert in the exact reverse order

Checklist:

- identify whether the forward logic is:
  - permute then xor
  - xor then permute
- read the exact permutation table
- read the exact target bytes
- write one short script to invert the operations in reverse order
- print the recovered input directly

Hard rule:

- do not guess operation order
- do not reconstruct the table from memory by hand if the exact array is readable

Expected evidence:

- exact permutation table
- exact target bytes
- exact recovered candidate

Fallback if fail:

- check dropped trailing indices
- check whether the same index is used for both reorder and xor
