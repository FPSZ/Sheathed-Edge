# CISCN Finals Web: MediaDrive

Challenge root:

- `D:\AWDP\CISCN半决赛\攻防\WEB\MediaDrive`

Blue goal:

- repair the exploitable edge with minimum changes
- keep upload, preview, download, and profile pages working

## Primary Risk Summary

Main risky edges:

- cookie deserialization:
  - `index.php`
  - `preview.php`
  - `profile.php`
  - `download.php`
- file preview path construction:
  - `preview.php`
- user-controlled encoding that affects path conversion:
  - `profile.php`

Most blue-safe repair order:

1. remove unsafe cookie deserialization
2. lock preview/download to the uploads directory using canonical paths
3. reduce encoding choices if still needed, but only if the first two patches are insufficient

## Minimal Patch Guidance

### Patch A: remove unsafe cookie object flow

Vulnerable edge:

- attacker-controlled `user` cookie
- sink: `unserialize()`

Minimum-change patch point:

- replace `@unserialize($_COOKIE['user'])` with a safe fixed user object or a narrow scalar-only cookie format

Preferred minimal direction:

- do not deserialize arbitrary objects from cookies
- if persistence is needed, store only a username string and rebuild a fresh `User`

Reject these larger changes:

- rewriting the whole auth system
- adding a new database-backed session layer during the match

### Patch B: canonicalize preview path after conversion

Vulnerable edge:

- `preview.php` builds `$rawPath = $user->basePath . $f`
- converted path is used in `file_get_contents()` without a `realpath` confinement check

Minimum-change patch point:

- after conversion, resolve `realpath`
- ensure the resolved path stays under the real uploads directory
- deny non-files

Preferred minimal direction:

- mirror the safer confinement style already used in `download.php`

Reject these larger changes:

- deleting preview functionality
- replacing preview with a different storage backend

### Patch C: reduce user-controlled encoding impact only if needed

Vulnerable edge:

- `profile.php` allows multiple encodings
- that choice later affects preview/download path conversion

Minimum-change patch point:

- narrow the allowed encoding set or freeze it to one safe value if exploitation still depends on alternate encodings

Preferred minimal direction:

- if business does not truly need alternate encodings, force `UTF-8`
- if compatibility is required, keep a tiny allowlist and pair it with canonical path checks

Reject these larger changes:

- removing profile page entirely
- redesigning all upload naming behavior

## Supervision Baseline

A good blue answer must say:

- the core bug is not "all file operations", but the exact source-to-sink chain
- the first priority is `unserialize` removal and preview path confinement
- larger rewrites are unnecessary during the match

A weak answer is wrong if it:

- only says "filter ../"
- only says "disable uploads"
- proposes a framework rewrite
- ignores the cookie deserialization issue

## Minimum Regression Check

- upload a normal allowed file and confirm it still appears in the file list
- preview a normal uploaded file and confirm it still opens
- download a normal uploaded file and confirm it still downloads
- send the previous exploit path and confirm it is blocked
