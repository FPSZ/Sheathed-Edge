# Web Family Skills

This document is the family skill reference for `task_family=web`.

Use it after the shared router has classified the task as `web`. Do not use it as the first step before routing.

## Stable Fields

When possible, keep these concepts visible in your reasoning:

- `task_family: web`
- `phase`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

## route-and-parameter-enumeration

Use this when the task has just entered `web` or the reachable attack surface is still unclear.

Goals:

- map routes, methods, parameters, and state inputs
- identify the smallest useful request surface
- avoid random payloading before the request model is clear

Checklist:

- identify route or endpoint candidates
- list user-controlled inputs
- note auth, cookie, header, and body dependencies
- separate confirmed inputs from guessed ones
- choose the next `primary_skill`

Expected evidence:

- route map
- input map
- state dependencies
- next target sink

Fallback if fail:

- if routing is unclear, step down to the smallest reproducible request
- if dynamic browsing is noisy, use server-side clues or static route evidence

## sink-and-filter-judgement

Use this after enumeration when the question becomes where input lands and how it is filtered.

Goals:

- locate sinks or sensitive code paths
- identify validation and normalization behavior
- tie every hypothesis to a concrete sink candidate

Checklist:

- identify likely execution, template, query, file, or auth sinks
- record filters, blacklist rules, encoders, and normalizers
- decide which sink deserves the next payload attempt

Expected evidence:

- sink candidate
- filter model
- rejected sink hypotheses
- next payload objective

Fallback if fail:

- if the sink is unclear, return to parameter tracing
- if filters are inconsistent, use smaller probes with clearer expected outcomes

## payload-selection-and-iteration

Use this when there is enough sink and filter evidence to justify active probing.

Goals:

- choose the smallest payload that can validate the hypothesis
- iterate with evidence, not with random spraying
- keep payload choice tied to the current sink model

Checklist:

- state the exact hypothesis the payload is testing
- choose one primary payload path
- record expected success, partial success, and failure signals
- keep iterations narrow and comparable

Expected evidence:

- payload goal
- expected signal
- observed response change
- next payload decision

Fallback if fail:

- if the signal is ambiguous, reduce payload complexity
- if the payload path collapses, return to sink judgement instead of brute forcing variants

## echo-error-and-state-analysis

Use this when the main signal comes from output differences, errors, redirects, sessions, or state changes.

Goals:

- interpret response differences correctly
- separate reflection, execution, and state mutation
- avoid overclaiming a bug from a weak echo

Checklist:

- compare success and failure responses
- track codes, body changes, redirects, session updates, and timing
- distinguish reflected input from actual sink impact
- note what the current signal proves and what it does not prove

Expected evidence:

- response delta summary
- state transition summary
- confidence level of the current hypothesis
- next verification step

Fallback if fail:

- if responses are too noisy, simplify the request and remove variables
- if state is unstable, re-run from a clean baseline

## minimal-patch-and-regression

Use this when the task includes code repair, hardening, or AWDP service continuity.

Goals:

- propose the smallest safe change
- preserve business behavior
- make the fix match the proven sink and filter issue

Checklist:

- state the exact vulnerable path
- propose the narrowest effective mitigation
- note user-visible regression risks
- define minimum verification after patching

Expected evidence:

- vulnerable path
- minimal patch
- regression risks
- verification targets

Fallback if fail:

- if the fix is too broad, reduce it to the true sink boundary
- if the vulnerability proof is still weak, do not over-design the patch
