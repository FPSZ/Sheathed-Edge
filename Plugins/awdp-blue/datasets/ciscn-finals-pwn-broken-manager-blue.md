# CISCN Finals PWN: broken_manager

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\PWN\broken_manager\broken_manager`

Blue goal:

- repair the exploitable binary edge with minimum behavioral impact
- keep the service process and protocol alive

Blue supervision rule:

- do not default to "rewrite the whole program"
- do not default to "add canary / recompile everything" without proving that is the smallest workable patch
- do not invent source files such as `main.c`, `server.c`, or `src/*.c` when the shipped challenge root only contains binaries and runtime libs
- describe the fix in binary terms:
  - menu state guard
  - size clamp
  - index guard
  - pointer lifetime guard
  - one call-site patch

Expected blue route:

1. identify the crash or primitive boundary
2. identify the narrowest fix point:
   - bounds check
   - size clamp
   - state validation
   - pointer lifetime guard
   - one call-site guard
3. keep the command/menu protocol unchanged
4. define one exploit regression and one normal-path regression

Current supervision note:

- this case is included for binary blue training
- exact patch point still requires binary-specific analysis trace
- acceptable first blue answer must still name:
  - vulnerable edge
  - minimal patch point
  - why not broader change
  - minimum regression check
- unacceptable blue answer patterns:
  - fabricated `main.c` / `manager.c`
  - generic `gets()/strcpy()` template with no binary-specific boundary
