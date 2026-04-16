# AWDP Red Skills

This document is the role skill reference for `competition_mode=awdp` and `awdp_role=red`.

Use it after the shared router has classified the task and `awdp-core` has stabilized the match context.

## Stable Fields

When possible, keep these concepts visible:

- `competition_mode: awdp`
- `awdp_role: red`
- `task_family`
- `shared_domain`
- `phase`
- `service_goal`
- `checker_safety_required`
- `primary_skill`
- `secondary_skills`
- `evidence_required`
- `fallback_if_fail`

## service-intake-and-fast-attack

Use this when the first red-team question is whether the target service is alive and which shortest attack path is worth testing.

Goals:

- confirm the target still responds
- identify the smallest attackable boundary
- distinguish a first-score path from a merely interesting bug idea

Checklist:

- confirm the service is reachable enough to justify attack work
- identify the narrowest exploit boundary
- state the first concrete signal that would prove score potential
- keep the next move small enough to replay quickly

Expected evidence:

- live service status
- exploit boundary candidate
- first proof target

Fallback if fail:

- if the service is unstable, downgrade to the smallest health or route check
- if the boundary is unclear, fall back to family-specific triage instead of guessing

## web-attack-or-binary-attack

Use this once the target boundary is known and the question becomes how to exploit it inside the active family.

Goals:

- translate family evidence into one attack path
- avoid mixing too many hypotheses at once
- keep the exploit objective explicit

Checklist:

- choose one primary exploit path
- state the family-specific preconditions
- define the exact signal that proves the path works
- reject weaker attack branches explicitly

Expected evidence:

- chosen exploit path
- exploit preconditions
- success signal

Fallback if fail:

- if the chosen path stalls, step back to the nearest family skill rather than widening blindly
- if the service changed, refresh the smallest live evidence first

## exploit-reuse-and-scaling

Use this after one exploit path is proven and the next question is whether it can be reused or parameterized.

Goals:

- separate a one-off exploit from a scalable exploit
- define invariants that matter across targets
- avoid overfitting to one host, one nonce, or one lucky state

Checklist:

- state the required invariant inputs
- identify what must be parameterized per target
- remove unnecessary local assumptions
- keep the batch logic as small as possible

Expected evidence:

- reusable exploit shape
- target-specific parameters
- known fragility points

Fallback if fail:

- if reuse assumptions are weak, keep the exploit as a single-target tool and stop claiming scale
- if host variance is large, capture the exact divergence before rewriting the exploit

## flag-harvest-and-submit

Use this when exploitation is already producing candidate flags or secrets.

Goals:

- normalize extraction
- avoid duplicate or malformed submissions
- keep exploit execution separate from flag handling

Checklist:

- define the exact flag extraction point
- normalize output format before submit logic
- de-duplicate repeated candidates
- record which evidence proves the candidate is not junk

Expected evidence:

- extraction point
- normalized flag candidate
- submission-ready output

Fallback if fail:

- if multiple candidates appear, define the next deciding check before spamming submits
- if extraction is noisy, reduce the exploit scope instead of widening scraping logic

## post-attack-regression-check

Use this after a successful attack when the next question is whether the service still behaves in a usable way.

Goals:

- confirm the exploit did not accidentally destroy the path you rely on
- preserve replayability
- identify whether the target moved into a different state after compromise

Checklist:

- rerun the smallest service health or functional check
- note whether the exploit is still replayable
- record any state drift that affects later attacks
- state whether additional caution is required before batch use

Expected evidence:

- post-attack service state
- replayability note
- next red-team decision

Fallback if fail:

- if the service became unstable, stop scaling and fall back to the last safe exploit proof
- if state drift is large, split the workflow into exploit and restore assumptions explicitly
