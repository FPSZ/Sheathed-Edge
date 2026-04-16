# Reverse Family Skills

This document is the family skill reference for `task_family=reverse`.

Use it after the shared router has classified the task as `reverse` and the `binary-core` layer has established the first binary evidence set.

If `competition_mode=awdp`, keep the reverse route technical and let `awdp-core` plus `awdp-red` / `awdp-blue` control the match behavior.

## Stable Fields

When possible, keep these concepts visible:

- `task_family: reverse`
- `shared_domain: binary`
- `competition_mode`
- `awdp_role`
- `phase`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

## key-function-recovery

Use this when the first reverse question is which function or basic path actually matters.

Goals:

- locate the most relevant functions
- connect strings, xrefs, imports, and entry flow
- reduce the search space to one or two high-signal targets

Checklist:

- identify the most suspicious strings, symbols, or xrefs
- map entry or caller flow toward likely validation logic
- nominate the next key function
- explain why weaker function candidates were rejected
- cite the actual tool step used to obtain those candidates
- if a full flag already appears in strings with no contradictory logic, stop here and promote it to extraction instead of forcing deeper analysis
- if multiple flag-like strings appear, explicitly rank them and justify which one deserves one more verification step

Expected evidence:

- key function candidate
- supporting strings or xrefs
- narrow next target

Fallback if fail:

- if names are stripped, pivot to strings and xrefs
- if strings are weak, pivot to imports and entry flow

## validation-and-check-tracing

Use this when the main task is to understand how input is checked or how success is decided.

Goals:

- trace the validation path
- identify compare points, expected constants, or branch conditions
- understand what must be true for success

Checklist:

- follow the input path into the check logic
- identify comparisons, branch conditions, or table lookups
- summarize the exact success condition
- state what evidence is still missing before final extraction
- ground the summary in at least one concrete function, xref, or decompile result
- if strings/functions are still too shallow, take one deeper step only: one xref, one chosen function detail, or one pseudocode view

Expected evidence:

- check routine summary
- decisive branches or constants
- missing evidence list

Fallback if fail:

- if tracing is noisy, reduce scope to one branch or compare site
- if the check is indirect, step back to callers and state flow

## algorithm-reconstruction

Use this when the decisive blocker is the transform or algorithm itself.

Goals:

- reconstruct the minimal algorithm needed to derive the answer
- separate signal from compiler noise
- turn logic into reproducible steps

Checklist:

- identify state, loops, tables, and per-byte or per-block operations
- rewrite the logic in plain language
- capture constants, order, and edge handling
- decide whether a tiny local script is the next best step
- make sure the reconstruction references real tool-visible logic rather than pure guesswork

Expected evidence:

- algorithm summary
- constants / tables
- reproducible transform steps

Fallback if fail:

- if the full algorithm is too large, isolate one round or one byte path first
- if compiler noise dominates, simplify via pseudocode and variable role mapping

## decode-and-transform-inversion

Use this when the problem is now about reversing a known transform.

Goals:

- invert the discovered transform
- confirm candidate plaintext / flag structure
- keep the inversion reproducible

Checklist:

- state the forward transform clearly
- derive the inverse step order
- test the smallest inversion path
- verify the output shape against challenge conventions

Expected evidence:

- inverse transform
- candidate plaintext or flag
- validation signal

Fallback if fail:

- if inversion is ambiguous, go back to the exact forward step order
- if outputs look close but wrong, inspect indexing and constant order first

## flag-and-key-extraction

Use this when enough evidence exists to derive the final answer.

Goals:

- produce the final flag or key
- connect the answer to the evidence chain
- avoid over-explaining beyond what proves correctness

Checklist:

- state the exact extraction path
- cite the decisive function / constant / transform clue
- provide the final answer
- note any remaining uncertainty if still partial
- name the real tool result that most directly supports the answer
- never leave a raw `<tool_call>` block or tool-planning text in the final answer
- if a near-miss candidate exists, do one last narrow normalization or mutation check before answering
- never stop at "best candidate" when the remaining gap is only one reversible transform, one character mutation, or one index-based inversion

Expected evidence:

- final flag / key candidate
- extraction chain
- confidence level

Fallback if fail:

- if extraction is still partial, return to the closest unresolved transform edge
- if multiple candidates exist, state the deciding check you still need
- if a candidate differs from the obvious string by common reverse mutations such as `o -> 0`, index xor, permutation, or custom base64 alphabet, validate that exact mutation before giving up

## anti-obfuscation-fallback

Use this when packing, junk control flow, symbol stripping, or opaque transforms are the blocker.

Goals:

- identify the exact blocker
- choose the narrowest workaround
- avoid turning the solve into an open-ended deobfuscation project

Checklist:

- name the blocker precisely
- choose the next smallest workaround (unpack, rename, trace one path, dump one layer)
- keep the solve focused on the target evidence

Expected evidence:

- blocker summary
- workaround choice
- next evidence target

Fallback if fail:

- if the workaround grows too wide, step back to the smallest high-signal path
- if runtime tracing is required, define exactly what must be observed

## writeup-finalization

Use this when the solve is finished or the user explicitly asks for a writeup / WP.

Goals:

- produce a plain, low-AI-tone reverse WP
- keep the writeup evidence-based and reproducible
- separate solved facts from any remaining uncertainty

Checklist:

- write in short, professional, non-first-person language
- do not use `我`, `我们`, or AI-style filler openings
- keep the structure compact: title, solve process, final flag
- cite concrete evidence such as strings, constants, branches, transforms, addresses, or script output
- if a solve script exists, include a runnable final script section rather than pseudocode
- use relative paths in WP examples when possible; avoid absolute `D:\...` paths in the body
- if any step was not directly verified, label that gap explicitly instead of fabricating detail

Expected evidence:

- compact writeup outline
- decisive evidence chain
- final flag or clearly labeled partial conclusion

Fallback if fail:

- if evidence is too thin, produce a short factual solve note instead of bloated prose
- if the solve is still partial, mark the WP as partial and state the exact unresolved gap


## micro-patterns-for-must-pass-devset

Use these compact patterns for small reverse tasks that should close quickly.

### custom-alphabet-and-encoding

Use when strings expose:
- a custom alphabet
- a ciphertext blob
- obvious base64/base32-like shape

Rules:
- first decide whether the binary is encoding or decoding
- if a custom alphabet is present, build an index mapping back to the standard alphabet before decoding
- do not stop at a plausible decoded string if one tiny inversion step can produce the exact flag

### runtime-mutation-of-static-candidate

Use when strings expose a near-flag like `{hello_world}` but the check function may mutate it before compare.

Rules:
- treat static candidate strings as pre-mutation candidates, not automatic final answers
- do one narrow function-detail or command step on the compare path
- explicitly test common reverse mutations such as `o -> 0`, case rewrite, index xor, add/sub by index

### permutation-then-byte-transform

Use when the challenge length is fixed and logic suggests both reordering and xor/add/sub by index.

Rules:
- determine the forward order first
- invert in the exact reverse order
- do not mix up "permute then xor" with "xor then permute"
- if the output length is wrong, check whether you dropped trailing indices before assuming the whole reconstruction is wrong

### packed-or-upx-early-stop

Use when filename, strings, or section/import clues strongly indicate UPX or a simple packer.

Rules:
- if strings already reveal the complete flag, stop and answer
- if strings are thin but UPX is strongly suggested, treat unpacking as the next narrow evidence step, not a whole open-ended reversing project
- do not spend long on ordinary function walkthroughs before deciding whether the packer is the real blocker
