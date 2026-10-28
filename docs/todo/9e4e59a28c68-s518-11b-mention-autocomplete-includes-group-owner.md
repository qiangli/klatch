---
id: 9e4e59a28c68
kind: bug
title: S518.11b @mention autocomplete includes group owner for members
seq: 2823
status: todo
priority: p0
labels:
    - klatch
created: 2026-10-28T10:22:21.46136Z
due: 2026-10-31T00:00:00Z
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
---

S518.11b @mention autocomplete must include the group OWNER for members (bug found by live E2E).
Repo: github.com/qiangli/klatch. Measured live (v1.2.2): in group 'S518 test' owned by alice with members bob and carol, bob types '@' in the composer and the autocomplete does NOT list @alice (members see the channel through alice's contact card; the channel's member/card list from a member's view excludes the owner). Fix in app/client/web: the mention candidate list for a channel = channel owner (the card/contact the channel came from, when the viewer is not the owner) + all other members + (for the owner) all cards, de-duplicated, excluding the signed-in user. Also make the 'mentioned you' badge/highlight logic use the same candidate set. Unit tests in src/utils/mentions.test.ts (and the hook if testable) for owner-view and member-view. Verification: yarn test, type-check, build green; commit with trailers; exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
