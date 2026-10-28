---
id: 59807504c625
kind: bug
title: 'S518.6h v1.2.3 regression: members do not see owner group in web UI'
seq: 2825
status: todo
priority: p0
labels:
    - klatch
created: 2026-10-28T11:01:39.921521Z
due: 2026-10-31T00:00:00Z
assignee: claude-sonnet5.5
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
---

S518.6h REGRESSION in v1.2.3: members no longer see the owner's group in the web UI (even after reload).
Repo: github.com/qiangli/klatch (main 51a928ea = v1.2.3, live on the dev satellite since ~06:00 PDT).
Measured by the manager: with v1.2.2 (web built against npm databag-client-sdk 0.0.44) the live E2E test 2 'alice posts in the group; bob and carol see it' PASSED (after one cache-clear reload). With v1.2.3 (S518.6f switched app/client/web to "databag-client-sdk": "file:../../sdk" and changed app/sdk/src/connection.ts + net/server/internal/api_status.go) it FAILS: carol's UI shows 'No Topics' even after reload. Server side is correct: carol's card for alice is status=connected with notifiedChannel=78, and GET /content/channels?contact=<alice-guid>.<token>&viewRevision=1 (the member pull) returns 200 with subject 'S518 test'. So the regression is in the client: the in-repo app/sdk at HEAD differs from the published 0.0.44 the web app was tested with (or the new connection.ts ordering breaks the initial card/channel sync).
Task: find the root cause and fix it with the smallest safe change. Acceptable outcomes: (a) fix the in-repo SDK path so members see owner channels and live revisions arrive, or (b) revert app/client/web to npm databag-client-sdk ^0.0.44 (restore yarn.lock), port the minimal connection fix into the web app if it is needed and possible, and keep the server-side api_status.go/notify fixes — whichever is proven. Prove it with the REAL E2E on the live host: the suite is at ~/s518-klatch/e2e on the novidesign host (ACCESS line: bashy claim list | grep s518; runner runit.sh; it needs env KLATCH_PASS_ALICE/BOB/CAROL (and DAVE + KLATCH2_URL for federation) — you can reach the vault from Dragon with bashy secret show klatch/<name> and klatch2/dave and pass them to the host over ssh stdin, never print them). To test your build on the live host WITHOUT a release, you may temporarily swap ONLY the web bundle of the 'klatch' service (keep a copy of the current ~/s518-klatch web dir and restore it before you exit unless your fix is better and documented); never touch data dirs or klatch2. Front rate limit ~60 req/min per IP applies to the E2E (it paces requests).
Verification: E2E tests 1-6 + 8 pass on the host with your web build; go test/web/sdk gates green; commit with trailers and the measured E2E summary; exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
