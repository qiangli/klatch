---
id: 4d205fe32b38
kind: bug
title: S518.6d Same-node notifications over loopback (front rate limit drops them)
seq: 2822
status: todo
priority: p0
labels:
    - klatch
created: 2026-10-28T09:46:19.921905Z
due: 2026-10-31T00:00:00Z
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
---

S518.6d Same-node contact notifications go over loopback, not through the public front (bug found by the S518.10 live E2E).
Repo: github.com/qiangli/klatch (this workspace). Problem (measured): the server delivers contact notifications (net/server/internal/notify.go: POST/PUT https://<node>/contact/...) even when the member's node is THIS node, so they leave the host and come back through the satellite front, which rate-limits anonymous callers to 60 req/min per source IP (measured from the host: 70 rapid GETs -> 56x200, 14x429). Under load the notifications get 429 and are dropped, so members (e.g. carol) intermittently never see a new group/message.
Fix: when notification.Node equals this node's configured federation domain (the admin config 'domain', may contain a path like '<sat>/matrix/h/<host>/app/klatch'), deliver to the local listener instead: http://<bind-address-from -p, e.g. 127.0.0.1:7100>/contact/<module>?contact=... (same method/body, no Origin/Bearer needed), or call the handler in-process if cleaner. Keep the existing public path for other nodes. Also log the HTTP status on any failed notify (today it only logs 'failed to notify contact') so 429s are visible. Unit tests: URL selection for same-node vs remote (domain with and without path, trailing slash), and that the -p host:port bind is used.
Local proof (Dragon, short): run two accounts on one local server behind your prefix proxy configured to 429 after N requests; show that a group post still reaches the member (notified_channel increments) while the proxy is limiting.
Verification: cd net/server && go test ./... green; web untouched; commit with trailers and the measured proof; exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
