---
id: 529c4c5bb55e
kind: feature
title: S518.3 fork qiangli/klatch from a pinned Databag tag and MVP rebrand with credits
seq: 2811
status: done
priority: p0
labels:
    - klatch
created: 2026-10-28T05:42:54.857995Z
weave: 3
due: 2026-10-31T00:00:00Z
assignee: claude-sonnet5.5
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
closed: 2026-10-28T06:18:07.332346Z
resolution: fixed
closed_by: claude-sonnet5.5-c
---

S518.3 Rebrand the Databag fork as Klatch (MVP) with full credit to the author.

Repo: this workspace is github.com/qiangli/klatch (fork of balzack/databag at 45343338). Scope: branding only — app/client/web (title, index.html, manifest/public icons+logo, visible "Databag" strings in UI/i18n, an About/Credits view or section reachable from settings), net/server user-visible strings (if any), README.md (top section), new NOTICE file. Do NOT touch app/client/web/src conversation/compose/message components (another worker adds @mention there), app/mobile, app/sdk, go.mod module paths, API routes, storage keys, or package names (keep them stable so upstream rebases stay easy).
Do:
- NOTICE: "Klatch is based on Databag by Roland Osborne (https://github.com/balzack/databag), licensed under the Apache License 2.0. Modifications Copyright 2026 DHNT Inc." Keep LICENSE and all copyright headers.
- Product name "Klatch" everywhere user-visible in the web app; simple original logo (SVG, e.g. a speech-bubble cluster, no Databag artwork) + favicon/PWA icons generated from it.
- About/Credits: version, "Based on Databag by Roland Osborne — Apache-2.0" with links to upstream + license.
- README top: Klatch intro + credit; keep upstream README below a heading "Upstream Databag README".
- Change notes: add a one-line "Modified by DHNT Inc. for Klatch" comment to each modified source file that has a header (Apache-2.0 §4b).
- Keep it ONE small commit stack (2-4 commits) that rebases cleanly onto upstream.
Verification: `cd app/client/web && yarn --frozen-lockfile && yarn type-check && yarn build` passes (paste last lines); grep shows no user-visible "Databag" left in app/client/web/src except credits/notice; remove node_modules and dist before finishing.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
