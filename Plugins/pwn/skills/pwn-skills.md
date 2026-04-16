# Pwn Family Skills

This document is the family skill reference for `task_family=pwn`.

Use it after the shared router has classified the task as `pwn` and the `binary-core` layer has established the initial binary evidence. Do not use it as the first step before routing.

If `competition_mode=awdp`, keep the pwn route focused on exploitability or binary patch boundaries, and let `awdp-core` plus `awdp-red` / `awdp-blue` control the match workflow.

## Stable Fields

When possible, keep these concepts visible in your reasoning:

- `task_family: pwn`
- `shared_domain: binary`
- `competition_mode`
- `awdp_role`
- `phase`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

## binary-intake-and-triage

Use this right after shared binary intake when the pwn-specific question is whether the target really contains an exploit-relevant boundary.

Goals:

- connect the binary surface to an exploit-relevant interaction point
- identify whether the next move is exploit judgement, debugger validation, or patch boundary review
- avoid jumping into exploit scripting from vague binary evidence

Checklist:

- identify the input boundary, protocol edge, or crash surface that actually matters
- distinguish validation logic from memory-corruption logic
- name the narrow next evidence target before selecting an exploit chain

Expected evidence:

- exploit-relevant boundary summary
- current uncertainty list
- narrow next target

Fallback if fail:

- if the boundary is unclear, go back to one narrower binary-core step
- if the service shape is unclear, collect one live interaction or protocol clue first

## mitigation-and-primitive-judgement

Use this after shared binary intake when the next question is exploitability, patchability, or bug-shape judgement.

Goals:

- determine relevant mitigations
- infer likely primitive
- avoid wasting time on exploit paths blocked by protections

Checklist:

- map protections to likely exploitation constraints
- classify the bug or primitive candidate
- decide whether the target supports stack, heap, format-string, logic, or loader-oriented paths
- record what is still missing before exploit work

Expected evidence:

- mitigation summary
- primitive candidate
- blocked paths
- candidate paths

Fallback if fail:

- if primitive is unclear, go back to narrower triage or debugger evidence
- if mitigation evidence conflicts, prefer the smallest reproducible proof

## exploit-path-selection

Use this when there is enough evidence to choose an exploitation route.

Goals:

- rank candidate exploit chains
- choose the shortest viable path
- avoid building an exp skeleton before the chain is justified

Checklist:

- list candidate paths and reject weak ones explicitly
- pick one primary chain
- note dependencies such as leak, overwrite, pivot, or one-shot conditions
- state what evidence is required before scripting

Expected evidence:

- chosen chain
- rejected alternatives
- required leak or write conditions
- next verification step

Fallback if fail:

- if the chosen path stalls, step back one layer and re-check primitive judgement
- if the environment differs from assumptions, switch to verification before rewriting the whole plan

## debug-and-failure-recovery

Use this when the exploit, run, or debugger path is failing.

Goals:

- identify why the current route failed
- preserve useful evidence
- recover without restarting the whole solve

Checklist:

- distinguish environment failure from reasoning failure
- check offsets, payload shape, timing, libc, and interaction assumptions
- keep a short failure log
- set a concrete `fallback_if_fail`

Expected evidence:

- failure point
- most likely cause
- next recovery step

Fallback if fail:

- if dynamic evidence is too noisy, go back to static confirmation
- if static evidence is insufficient, create the smallest reproducible run

## minimal-patch-and-regression

Use this when the task includes patching, hardening, or AWDP service continuity.

Goals:

- propose the smallest safe patch
- avoid breaking service behavior
- keep exploit and fix reasoning aligned

Checklist:

- identify the real root cause
- propose the narrowest effective fix
- note expected regression surface
- define minimum verification after patching

Expected evidence:

- root cause
- minimal fix
- regression risks
- verification targets

Fallback if fail:

- if the patch is too invasive, step back and isolate the dangerous edge
- if exploit evidence is weak, do not overspecify the fix
