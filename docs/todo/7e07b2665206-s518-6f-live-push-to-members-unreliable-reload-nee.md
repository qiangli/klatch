---
id: 7e07b2665206
kind: bug
title: S518.6f Live push to members unreliable (reload needed)
seq: 2824
status: done
priority: p0
labels:
    - klatch
created: 2026-10-28T10:22:21.567645Z
weave: 9
due: 2026-10-31T00:00:00Z
assignee: codex-gpt6-sol
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
closed: 2026-10-28T10:32:58.061224Z
resolution: fixed
closed_by: codex-gpt6-sol-i
---

S518.6f Live push: members sometimes see new group messages/DMs only after a reload (found by live E2E on v1.2.2).
Repo: github.com/qiangli/klatch. Deployment: node on 127.0.0.1:7100 behind the satellite front at https://<sat>/matrix/h/<host>/app/klatch/ (outpost proxy strips the prefix, proxies websockets; front rate-limits anon traffic ~60/min/IP; same-node notifications now go over loopback since v1.2.2). Symptom: E2E notes 'carol did not receive "S518 test" by push; clearing the local client cache and reloading' and same for a DM to bob — after reload the content is there, so storage is right but the live update path is unreliable. Investigate the live path end to end: server -> client websocket (/status endpoint, api_status.go; message format, revision pushes), the web client's websocket URL under a path prefix (does it include the mount path? reconnect/backoff? does a rate-limited (429) websocket upgrade or a reconnect storm happen?), and revision propagation for cards/channels of OTHER owners (member views of an owner's channel are pulled via the contact token after a card revision change — check the server bumps and pushes the member's card revision when the owner's channel changes). Reproduce locally with two accounts behind your prefix proxy (strip prefix, proxy websocket, optionally 429 after N req/min) and fix the root cause in server and/or web client. Tests where possible. Verification: go test ./... + web yarn test/type-check/build green; local proof that a member receives a new group message and a DM live without reload; commit with trailers; exit. ACCESS to the live host is available (bashy claim list | grep s518) for read-only diagnosis (logs), do not redeploy.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
