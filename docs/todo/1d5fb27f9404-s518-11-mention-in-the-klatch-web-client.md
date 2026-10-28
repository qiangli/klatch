---
id: 1d5fb27f9404
kind: feature
title: S518.11 @mention in the Klatch web client
seq: 2819
status: done
priority: p0
labels:
    - klatch
created: 2026-10-28T05:42:56.951063Z
weave: 4
due: 2026-10-31T00:00:00Z
assignee: codex-gpt5.6-sol
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
closed: 2026-10-28T06:39:56.592435Z
resolution: fixed
closed_by: codex-gpt5.6-sol-d
---

S518.11 @mention in the Klatch web client (app/client/web).

Repo: this workspace is github.com/qiangli/klatch (Databag fork). Upstream has no @mention. Scope: app/client/web/src conversation/compose/message rendering + channel list badge, plus their tests. Do NOT touch branding files (index.html, public/, README, NOTICE, About) — another worker rebrands in parallel. Do not change app/sdk or the server; implement client-side so it works in sealed (E2E) topics.
Behavior:
1. Typing "@" in the message composer opens an autocomplete of the current channel's members (handle + display name); arrow/enter/click inserts a mention token "@handle".
2. Rendered messages highlight mention tokens; a mention of the signed-in user is highlighted more strongly.
3. Channel list / topic list shows a "mentioned you" badge (e.g. "@" marker) on channels with unread messages that mention the signed-in user; cleared when the channel is opened/read.
4. Mention parsing in one small pure module (e.g. src/utils/mentions.ts) with unit tests (jest is configured): parse, match members, edge cases (email addresses are not mentions, punctuation, unknown handles).
Verification: `cd app/client/web && yarn --frozen-lockfile && yarn test mentions` (or jest path) green, `yarn type-check` and `yarn build` pass — paste results in the commit message. Remove node_modules and dist before finishing.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
