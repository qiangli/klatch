---
id: a02c87554fba
kind: bug
title: 'S518.6a Klatch server/web: base path, loopback bind, Go-native image transforms'
seq: 2820
status: done
priority: p0
labels:
    - klatch
created: 2026-10-28T06:48:04.544794Z
weave: 5
due: 2026-10-31T00:00:00Z
assignee: codex-gpt6-sol
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
closed: 2026-10-28T08:23:39.580858Z
resolution: fixed
closed_by: s518-manager
---

S518.6a REWORK (bug found by S518.10 live E2E): the stock Klatch web login gets 401 through the satellite because the front strips Authorization and the web app uses the PUBLISHED npm package databag-client-sdk (app/client/web/package.json "databag-client-sdk": "^0.0.44"), not this repo's app/sdk, so your earlier SDK change never reaches the browser.
Fix in app/client/web (minimal, rebaseable): install a tiny fetch wrapper at app start (e.g. src/main.tsx or a new src/utils/credentialsFetch.ts imported first) that, for requests to the app's own origin, copies any `Authorization: Basic ...` header into `Credentials: Basic ...` (keep Authorization too); also cover XMLHttpRequest if the SDK uses it for login (check the npm SDK code in node_modules/databag-client-sdk). Unit test the wrapper (jest). Do not switch the dependency to the local sdk in this rework.
Verify locally as before with your prefix proxy that strips Authorization: a real browser-equivalent login through the web UI code path works (or a jest/jsdom test proving the header copy for the SDK's login call), plus yarn test, type-check, build green. Commit with trailers; then exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
