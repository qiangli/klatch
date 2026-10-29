---
id: 640c63990d4e
kind: bug
title: S518.11c @mention caret jumps back after selecting a mention
seq: 2826
status: todo
priority: p0
labels:
    - klatch
created: 2026-10-29T04:00:46.969182Z
due: 2026-10-31T00:00:00Z
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
---

S518.11c @mention: caret jumps back after selecting a mention (bug found by live E2E on v1.2.4).
Repo: github.com/qiangli/klatch (main eab650e4 = v1.2.4). Measured live: bob types '@' in the composer, the autocomplete lists '@alice' (selected), he presses Enter (same path as click: insertMention -> text '@alice ', caret 7), then types ' hi mvt0ind5' char by char. Expected value '@alice hi mvt0ind5'; actual '@alice mvt0ind5 hi ' — the caret is being reset back to the stored post-insertion caret while typing continues (likely a selection/caret restore in a useEffect or on every render/state update in app/client/web/src/conversation/Conversation.tsx / useConversation.hook.ts). Fix: restore the caret ONCE immediately after the insertion (e.g. requestAnimationFrame after the state update, then clear the pending caret), never on later renders; also ensure Enter/Tab/click all use the same path and that Enter while the list is open does not send the message. Add a test (jsdom/RTL or a hook-level test) that inserts a mention then types more characters and asserts the final value and caret.
LIVE PROOF (required, ~10 min): build app/sdk then app/client/web and temporarily swap ONLY the web bundle of the 'klatch' service on the novidesign host (ACCESS: bashy claim list | grep s518; back up ~/s518-klatch/data/web first), run the E2E suite there (~/s518-klatch/e2e; runit.sh; pass KLATCH_PASS_ALICE/BOB/CAROL/DAVE + KLATCH2_URL via ssh stdin from Dragon: bashy secret show klatch/<name>, klatch2/dave — never print). Target: tests 1-6 and 8 pass. Restore the original web bundle afterwards.
Verification: web yarn test/type-check/build green; E2E summary in the commit message; commit with trailers; exit.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
