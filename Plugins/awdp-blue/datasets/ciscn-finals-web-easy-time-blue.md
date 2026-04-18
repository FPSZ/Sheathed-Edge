# CISCN Finals Web: easy_time

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\WEB\easy_time\easy_time`

Blue goal:

- keep login, dashboard, board, avatar upload, and plugin upload alive
- repair the smallest exploit edge first

## Primary Risk Summary

Main risky edges visible in the current code:

- plugin zip extraction:
  - route: `/plugin/upload`
  - handler: `safe_upload()`
- secret management / cookie trust:
  - Flask `secret_key` is hard-coded and weak
- remote avatar fetch path:
  - `fetch_remote_avatar_info()`
  - currently reads arbitrary remote content and returns snippets

Most blue-safe repair order:

1. fix plugin upload extraction
2. reduce avatar fetch exposure if that route is used as an info leak or SSRF helper
3. harden secret and cookies without changing login flow

## Minimal Patch Guidance

### Patch A: replace unsafe zip extraction helper

Vulnerable edge:

- uploaded zip file
- sink: `safe_upload()` writes `info.filename` directly with `os.path.join()`

Minimum-change patch point:

- stop using `safe_upload()`
- route plugin upload through the already safer `safe_extract_zip()` path

Preferred minimal direction:

- in `/plugin/upload`, replace:
  - `extracted = safe_upload(saved, dest)`
- with:
  - `extracted = safe_extract_zip(saved, dest)`

Why this is the right blue fix:

- the safer extraction helper already exists
- it already checks absolute paths and ZipSlip traversal
- this blocks the proven exploit with one local change

Reject these larger changes:

- deleting plugin upload entirely without evidence the checker does not use it
- redesigning the plugin format
- rewriting the whole Flask app

### Patch B: keep avatar fetch from becoming a dangerous helper

Vulnerable edge:

- user-supplied `avatar_url`
- sink: outbound fetch plus returned content snippet

Minimum-change patch point:

- only fetch if hostname passes public-IP checks
- return metadata only, not arbitrary content snippets, if content leakage is unnecessary

Preferred minimal direction:

- preserve page rendering
- shrink returned remote data

Reject these larger changes:

- removing the about page
- deleting remote avatar support without checking service expectations

### Patch C: harden auth material without changing the route contract

Vulnerable edge:

- static weak Flask secret
- cookie-based trust path

Minimum-change patch point:

- move the secret to environment or a deployment secret
- keep the same route and cookie contract

Reject these larger changes:

- rewriting login storage
- adding a new user system during the match

## Supervision Baseline

A good blue answer must say:

- the best first fix is not a broad refactor
- the smallest repair is to stop using `safe_upload()` and switch to `safe_extract_zip()`
- this is a direct minimum-change patch because the safe helper already exists

A weak answer is wrong if it:

- only says "validate zip"
- suggests deleting the plugin route first
- ignores the existing safe helper

## Minimum Regression Check

- login still works
- plugin zip upload with a normal benign archive still succeeds
- a traversal zip is rejected
- board/about/dashboard still render
