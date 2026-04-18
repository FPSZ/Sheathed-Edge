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

## evidence-first-forensics

Use this when the blue-team task is still ambiguous and the next need is evidence collection, not immediate patching.

Goals:

- collect the minimum evidence set before touching files or binaries
- separate confirmed attacker behavior from guesses
- preserve artifacts needed for later replay, rollback, or reporting

Checklist:

- identify what artifact class exists:
  - process / port
  - log / config
  - uploaded file / webshell
  - binary diff
  - packet capture / pcap
- state which artifact is primary and why
- record the smallest evidence-producing command or read step
- avoid editing, deleting, or restarting the target before evidence is captured when that evidence may disappear

Expected evidence:

- artifact class
- exact evidence target
- smallest safe acquisition step

Fallback if fail:

- if the evidence source is volatile, capture summary indicators first
- if the source is already gone, pivot to adjacent logs, process state, or surviving network traces

## tshark-pcap-triage

Use this when AWDP blue work involves `.pcap`, `.pcapng`, live capture notes, suspicious egress, credential leakage, or checker traffic reconstruction.

Goals:

- extract attacker actions and service-critical flows quickly
- turn packet evidence into concrete patch or containment guidance
- avoid drowning in full-pcap output

Hard rule:

- do not dump the entire pcap by default
- always narrow by one of:
  - host
  - port
  - protocol
  - http request path
  - tcp stream
  - time window

Preferred tshark progression:

1. identify conversations:
   - `tshark -r <pcap> -q -z conv,tcp`
   - `tshark -r <pcap> -q -z conv,udp`
2. identify endpoints:
   - `tshark -r <pcap> -q -z endpoints,ip`
3. narrow to suspicious protocol:
   - `tshark -r <pcap> -Y "http || dns || tcp.port==<port>"`
4. extract decisive fields only:
   - `-T fields -e ip.src -e ip.dst -e tcp.dstport -e http.host -e http.request.uri`
5. if needed, inspect one stream:
   - `tshark -r <pcap> -qz follow,tcp,ascii,<stream_id>`

Local workstation note:

- on this machine, if `tshark` is not in PATH, use:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe`
- the GUI path is:
  - `D:\CTF\tool\zhuabao\Wireshark\Wireshark.exe`

Checklist:

- identify suspicious peer and target service
- identify exploit or beacon protocol
- identify whether credentials, paths, payloads, or webshell routes appear
- summarize exactly how packet evidence changes the patch boundary

Expected evidence:

- suspicious endpoint pair
- protocol or stream id
- decisive field extract
- resulting patch or containment implication

Fallback if fail:

- if tshark is unavailable, state that explicitly and fall back to:
  - file presence check
  - install path note
  - alternative parser only if already present
- if the pcap is too large, narrow by endpoints first before deeper parsing

## incident-artifact-cleanup-boundary

Use this when blue work involves removing webshells, rogue tasks, dropped binaries, or attacker persistence after evidence is already captured.

Goals:

- clean only what is justified by evidence
- keep the service alive and the checker path intact
- separate containment from destructive cleanup

Checklist:

- identify confirmed malicious artifact
- identify evidence already preserved
- choose the smallest cleanup action:
  - quarantine
  - rename
  - permission removal
  - route disable
  - targeted deletion
- define one post-cleanup service check

Expected evidence:

- confirmed artifact
- cleanup action
- post-cleanup verification

Fallback if fail:

- if artifact confidence is weak, quarantine instead of deleting
- if cleanup may break service, contain first and defer full removal

## hotfix-and-minimal-patch

Use this when the vulnerable boundary is known and the next question is the smallest workable fix.

Goals:

- isolate the smallest patch boundary
- preserve existing service behavior
- keep rollback and replay practical

Hard rule:

- blue-team fixes should follow the minimum-change law
- prefer a local guard, whitelist, canonicalization step, or one-line safe API replacement over structural rewrites
- do not widen the patch across unrelated routes, templates, or modules unless the exploit chain truly crosses them
- if the challenge only ships binaries and libc/loader artifacts, do not invent source files such as `main.c`, `server.c`, or controller names that are not present
- in binary-only cases, describe the patch boundary as:
  - menu state
  - size field
  - index check
  - pointer lifetime guard
  - function boundary
  - or binary patch location

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

## minimum-change-law

Use this when there are multiple possible fixes and the blue-team question is which one is match-safe.

Goals:

- block the proven exploit chain with the fewest risky edits
- keep service behavior and checker behavior stable
- avoid turning a hotfix into a refactor

Checklist:

- identify the exact attacker-controlled input
- identify the exact dangerous sink
- patch the closest reliable boundary between them
- prefer:
  - input validation
  - strict allowlists
  - canonical path checks
  - parameterized queries
  - safe deserialization removal
  - permission or route narrowing
- reject:
  - framework migration
  - multi-module redesign
  - "cleaner" rewrites that touch unrelated code
  - hallucinated source trees that are not present in the shipped challenge files

Expected evidence:

- exact source-to-sink edge
- smallest effective patch point
- rejected larger alternatives

Fallback if fail:

- if several patch points look valid, choose the one closest to the sink that least changes normal behavior
- if the sink is still unclear, return to family-level evidence before patching

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

## blue-supervision-output

Use this when reviewing a model-produced AWDP blue answer or training sample.

Goals:

- judge whether the fix is actually minimal
- judge whether it blocks the stated exploit path
- judge whether it preserves the expected service path

Checklist:

- require these four items in the review:
  - vulnerable edge
  - minimal patch point
  - why larger edits are unnecessary
  - smallest regression check
- mark the answer incorrect if it:
  - patches the wrong layer
  - rewrites unrelated code
  - claims "fixed" without naming the exploit path
  - ignores checker or availability risk

Expected evidence:

- pass/fail supervision note
- one-line reason
- corrected minimal patch direction if needed

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
