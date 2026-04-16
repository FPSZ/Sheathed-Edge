# AWDP Blue Skills

This document is the role skill reference for `competition_mode=awdp` and `awdp_role=blue`.

Use it after the shared router has classified the task and `awdp-core` has stabilized the match context.

## Stable Fields

When possible, keep these concepts visible:

- `competition_mode: awdp`
- `awdp_role: blue`
- `task_family`
- `shared_domain`
- `phase`
- `service_goal`
- `checker_safety_required`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

## service-intake-and-risk-triage

Use this when the first blue-team question is which service edge is most dangerous right now.

Goals:

- confirm whether the service is alive enough to patch safely
- identify the highest-risk boundary
- avoid widening the fix scope before the real root cause is known

Checklist:

- confirm current service state
- identify the most dangerous reachable boundary
- note what business path must remain stable
- define the smallest next evidence step before patching

Expected evidence:

- service status
- highest-risk boundary
- protected business path

Fallback if fail:

- if the service is unstable, reduce to a health-first recovery step
- if risk is unclear, return to the family skill that best narrows the vulnerable edge

## hotfix-and-minimal-patch

Use this when the vulnerable boundary is known and the next question is the smallest workable fix.

Goals:

- isolate the smallest patch boundary
- preserve existing service behavior
- keep rollback and replay practical

Checklist:

- state the exact vulnerable edge
- choose the smallest effective fix
- reject broader patch ideas explicitly
- note what must be rechecked immediately after the change

Expected evidence:

- vulnerable edge summary
- minimal fix proposal
- immediate recheck list

Fallback if fail:

- if the patch grows across unrelated files or routes, step back and narrow the boundary again
- if root cause is weak, stop before overspecifying the fix

## checker-safe-fix

Use this when the patch may affect the checker or any score-critical service behavior.

Goals:

- identify the checker-critical contract
- keep the fix aligned with that contract
- avoid passing security review while failing the match

Checklist:

- identify route, method, response, protocol, or state assumptions that the checker likely depends on
- state what must remain unchanged
- define what may change safely
- keep the patch boundary narrower than the checker contract

Expected evidence:

- checker-critical contract
- safe change boundary
- unsafe change list

Fallback if fail:

- if the checker contract is unclear, infer the smallest observable contract first instead of guessing broadly
- if the patch would clearly alter the contract, find a narrower mitigation

## availability-preserving-defense

Use this when the main risk is that the fix could slow, crash, deadlock, or otherwise destabilize the service.

Goals:

- preserve service availability while reducing exploitability
- identify operational fragility before shipping the fix
- keep the defense proportional to the service shape

Checklist:

- note any performance, crash, timeout, or state risks introduced by the fix
- prefer guardrails that fail closed without killing normal requests
- define one minimal runtime or functional check after patching

Expected evidence:

- availability risk summary
- stability-preserving choice
- post-patch functional check

Fallback if fail:

- if the defense path looks too invasive, retreat to the smallest guard that blocks the proven exploit edge
- if availability signals are weak, do not claim service safety yet

## patch-regression

Use this after a patch exists and the next question is whether the service still works and the bug is actually constrained.

Goals:

- confirm the vulnerable path is reduced or blocked
- confirm the service still behaves correctly
- keep regression proof compact and replayable

Checklist:

- test the smallest exploit or vulnerability proof against the patch
- test the smallest normal service path that must still work
- state what remains unverified instead of pretending full coverage
- record rollback conditions if the patch is still risky

Expected evidence:

- blocked or reduced exploit path
- preserved normal behavior
- remaining uncertainty list

Fallback if fail:

- if the regression surface is too wide, test the two most score-critical behaviors first
- if the patch breaks normal behavior, revert to the minimal boundary review rather than layering more fixes

## backdoor-hunt-and-cleanup

Use this when there is evidence of webshells, tampered binaries, rogue accounts, cron jobs, or persistent attacker artifacts.

Goals:

- identify malicious artifacts precisely
- clean only what is justified by evidence
- avoid deleting legitimate service state or tooling

Checklist:

- name the artifact class and exact indicator
- distinguish suspicious from confirmed malicious material
- plan the smallest safe cleanup step
- note any service path that could break if the artifact guess is wrong

Expected evidence:

- artifact summary
- cleanup boundary
- post-cleanup verification target

Fallback if fail:

- if artifact confidence is weak, quarantine or narrow the evidence instead of deleting broadly
- if cleanup risk is high, separate containment from removal
