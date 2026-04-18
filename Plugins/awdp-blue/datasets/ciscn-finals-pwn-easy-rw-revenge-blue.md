# CISCN Finals PWN: easy_rw_revenge

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\PWN\easy_rw_revenge\easy_rw_revenge`

Blue goal:

- constrain the read/write primitive with the smallest patch
- keep the service protocol and process behavior stable

Supervision rule:

- prefer:
  - index bounds checks
  - size checks
  - pointer validity checks
  - menu state guards
- reject:
  - full binary rewrite
  - feature deletion without proof
- do not invent `main.c`, `server.c`, or `src/*.c`
- because this case likely centers on a read/write primitive, answers should prefer:
  - exact read/write size clamp
  - slot/index range guard
  - one-time state guard
  - pointer/null guard

Current supervision note:

- reverse trace already shows this is not a generic stdin overflow toy
- the binary exposes an RTSP / JSON command protocol with:
  - `add`
  - `delete`
  - `edit`
  - `show`
- string evidence explicitly shows:
  - `ADD_CHECK_PARAM`
  - `DEL_CHECK_INDEX`
  - `EDIT_CHECK_INDEX`
  - `EDIT_CONTENT_LARGE`
  - `Invalid show index`
- therefore acceptable blue answers should stay near:
  - slot/index bounds
  - content length clamp
  - add/edit/show state guards
- unacceptable answer pattern:
  - broad stack-overflow template
  - generic `gets()` replacement
  - invented `main.c` / `server.c`
  - answers that fail to mention the command/index/content boundary
