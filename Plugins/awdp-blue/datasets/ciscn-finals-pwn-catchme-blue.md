# CISCN Finals PWN: catchme

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\PWN\catchme`

Blue goal:

- preserve service behavior
- block the smallest proven exploit primitive

Supervision rule:

- hotfixes should prefer one local binary patch or one narrow source fix
- avoid protocol redesign during the match
- if no source files are shipped, do not hallucinate them
- acceptable patch language is binary-oriented:
  - exact function boundary
  - input length clamp
  - index range check
  - pointer/state guard

Current supervision note:

- included as a blue binary repair case
- exact sink and patch point require further binary triage
