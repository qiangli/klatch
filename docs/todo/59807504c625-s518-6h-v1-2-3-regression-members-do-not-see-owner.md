---
id: 59807504c625
kind: bug
title: 'S518.6h v1.2.3 regression: members do not see owner group in web UI'
seq: 2825
status: done
priority: p0
labels:
    - klatch
created: 2026-10-28T11:01:39.921521Z
weave: 11
due: 2026-10-31T00:00:00Z
assignee: claude-sonnet5.5
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
closed: 2026-10-29T02:41:31.308256Z
resolution: fixed
closed_by: claude-sonnet5.5-k
---

S518.6h FIX (operator: fix remaining issues). Root cause is MEASURED (see umbrella docs/sprint-518/e2e/BLOCKERS.md and your earlier S518.6h BLOCKERS diagnosis): behind the satellite front, anonymous traffic is rate-limited (~60 req/min per source IP; 429 with Retry-After; queued requests are delayed). The client SDK (app/sdk) then (1) hits its 15s fetch TIMEOUT (app/sdk/src/net/fetchUtil.ts) on contact channel/card syncs and (2) marks the contact 'offsync' (offsync_channel_card) and NEVER re-syncs it (contact.ts offsync branch only records the revision) — so members see 'No Topics' even after reload. A single unpaced user works.
Repo: github.com/qiangli/klatch (main = v1.2.3; web builds against in-repo app/sdk since v1.2.3).
Fix in app/sdk (and web only if needed), minimal and tested:
 a) fetchUtil: on HTTP 429 honour Retry-After (cap ~30s) and retry with jittered exponential backoff (bounded attempts); treat a timeout as retryable a bounded number of times; raise the per-request timeout to e.g. 30s.
 b) contact sync: an offsync contact must be retried — on the next revision event AND on a periodic timer (e.g. every 30-60s while offsync) — until the sync succeeds; clear the offsync marker on success; persisted offsync state must not block re-sync after reload/relogin.
 c) keep request volume modest (no tight polling).
Unit tests (jest in app/sdk) for 429/Retry-After retry, timeout retry, and offsync -> retried -> cleared.
LIVE PROOF (required): build the web bundle (cd app/sdk && yarn --frozen-lockfile && yarn build; cd app/client/web && yarn --frozen-lockfile && yarn build) and temporarily swap ONLY the web bundle of the 'klatch' service on the novidesign host (ACCESS: bashy claim list | grep s518; back up ~/s518-klatch/data/web first; never touch data or klatch2), then run the real E2E suite there (~/s518-klatch/e2e, runit.sh; pass KLATCH_PASS_ALICE/BOB/CAROL/DAVE + KLATCH2_URL via ssh stdin from Dragon vault: bashy secret show klatch/<name>, klatch2/dave — never print). Target: tests 1-6 and 8 pass (federation 7 optional) in one run with the default pacing. Afterwards RESTORE the original web bundle (the manager will ship your fix as a signed release).
Verification: sdk + web unit tests, type-check, builds green; E2E summary from the host in the commit message; commit with trailers; exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
