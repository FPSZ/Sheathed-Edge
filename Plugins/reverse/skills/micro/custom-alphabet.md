# Reverse Micro Skill: Custom Alphabet

Use this when strings or static data expose:

- a custom alphabet
- a ciphertext blob
- a base64/base32-like shape

Goal:

- map the custom alphabet back to the standard alphabet
- decode or invert with one tiny script

Checklist:

- decide first whether the binary is encoding or decoding
- read the exact alphabet
- read the exact ciphertext or target string
- do not hand-calculate the mapping
- immediately use one tiny script to:
  - build the translation table
  - decode the candidate
  - print the exact result

Expected evidence:

- exact custom alphabet
- exact ciphertext / target string
- exact decoded candidate

Fallback if fail:

- verify alphabet length
- verify whether only letters were rotated or the full alphabet was permuted
