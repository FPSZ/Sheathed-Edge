# CISCN Finals PWN: minidb

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\PWN\minidb\minidb`

Blue goal:

- keep the DB-like service logic alive
- hotfix the narrow memory-corruption or parser edge

Supervision rule:

- the model must not jump to "replace the database implementation"
- the fix should stay near:
  - parser length checks
  - object count checks
  - entry bounds
  - heap lifetime or double-free guards
- do not invent missing source files
- binary-blue language should stay near parser/menu/object/heap boundaries, not generic web/db redesign language

Current supervision note:

- included as a blue binary repair case
- exact patch point requires binary-specific trace
