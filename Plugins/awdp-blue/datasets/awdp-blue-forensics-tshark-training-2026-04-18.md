# AWDP Blue Forensics + tshark Training Notes

This note is for blue-team AWDP work where the model must prioritize evidence capture before patching.

## Core rule

- first preserve or read evidence
- then narrow attacker behavior
- then choose the smallest safe patch or cleanup step

## Packet-capture triage discipline

If a challenge directory contains `.pcap`, `.pcapng`, or capture notes, the model should not jump straight to "there is a webshell" or "block this IP" without packet evidence.

On this workstation, use the real binary path when PATH does not contain tshark:

- `D:\CTF\tool\zhuabao\Wireshark\tshark.exe`

Do not falsely claim `tshark` is unavailable if this path exists.

Use the smallest high-signal tshark workflow:

- identify top conversations:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe -r sample.pcap -q -z conv,tcp`
- identify top endpoints:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe -r sample.pcap -q -z endpoints,ip`
- identify likely exploit protocol:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe -r sample.pcap -Y "http || dns || tcp.port==8080"`
- extract only key fields:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe -r sample.pcap -Y "http.request" -T fields -e frame.number -e ip.src -e ip.dst -e http.host -e http.request.method -e http.request.uri`
- inspect only the suspicious stream:
  - `D:\CTF\tool\zhuabao\Wireshark\tshark.exe -r sample.pcap -qz follow,tcp,ascii,<stream_id>`

## What counts as good blue use of tshark

- names the suspicious endpoint pair
- names the suspicious route, payload, or credential indicator
- maps packet evidence to one concrete patch boundary
- keeps the service and checker path in mind

## What counts as bad blue use of tshark

- dumping the whole pcap with no narrowing
- claiming a webshell or exploit path without a route, stream, or payload indicator
- recommending firewall-only containment when the real issue is still an application bug
- deleting files before preserving packet-derived evidence

## Patch implications from packet evidence

Packet evidence should change one of these:

- the exact vulnerable route to patch
- the exact parser boundary to validate
- the exact credential or token to rotate
- the exact file write or upload primitive to contain
- the exact cleanup target to quarantine

## Minimum output shape for blue packet forensics

- `phase`: evidence-first-forensics
- `primary_skill`: `tshark-pcap-triage` or `evidence-first-forensics`
- `evidence_required`: pcap endpoint pair / stream / route / payload
- `minimal_patch_point`: exact route, parser, upload sink, or auth boundary
- `minimum_regression_check`: one live request or checker-safe behavior proof

## If tshark is missing

Do not hallucinate packet contents.

State one of:

- `tshark not installed; cannot claim packet-derived attacker behavior yet`
- `pcap exists but tshark is unavailable; preserve file and pivot to adjacent logs or artifact evidence`

Then continue with the best remaining evidence source.

Important:

- before saying `tshark` is unavailable, check the fixed local path above
