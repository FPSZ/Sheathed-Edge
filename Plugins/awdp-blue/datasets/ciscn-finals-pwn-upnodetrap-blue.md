# CISCN Finals PWN: UpNodeTrap

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\PWN\UpNodeTrap`

Blue goal:

- preserve the exposed service and front-end behavior
- repair the smallest binary exploitation edge

Supervision rule:

- if the service mixes a web/static layer with a native binary, do not rewrite the web layer first unless the exploit path clearly starts there
- prefer the closest native patch point that blocks exploitation
- do not fabricate source filenames when the shipped tree is binary-only plus front-end assets
- acceptable patch framing:
  - native function boundary
  - length/index guard
  - state transition guard
  - pointer guard

Current supervision note:

- included as a mixed-shape blue case
- exact exploit boundary still requires binary trace before semantic supervision can tighten
