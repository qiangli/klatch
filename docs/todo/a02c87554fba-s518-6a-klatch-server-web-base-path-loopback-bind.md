---
id: a02c87554fba
kind: bug
title: 'S518.6a Klatch server/web: base path, loopback bind, Go-native image transforms'
seq: 2820
status: todo
priority: p0
labels:
    - klatch
created: 2026-10-28T06:48:04.544794Z
due: 2026-10-31T00:00:00Z
sprint: 518
sprint_id: bbe9153b-b299-5be8-8ef8-215ad002deb9
sprint_title: 'Klatch MVP: rebranded Databag chat server + web + federation on one satellite (replaces Matrix), due Sat 10-31'
---

S518.6a Klatch server/web: serve under a URL base path, loopback bind, Go-native image transforms (deploy blockers found by the S518.2 spike).

Repo: this workspace is github.com/qiangli/klatch (Databag fork; already rebranded + @mention). Deployment model (decided by the sprint manager): Klatch runs on a bashy outpost host as an app on 127.0.0.1:<port>; the cloudbox satellite publishes it at https://<satellite>/matrix/h/<host>/app/klatch/ . The outpost proxy STRIPS the /matrix/h/<host>/app/klatch prefix before forwarding (requests reach the server at root paths like /account, /contact, /content, /admin, /status websocket) and sets header X-Forwarded-Prefix=/matrix/h/<host>/app/klatch . Websockets are proxied. Read net/server/main.go, net/server/internal (notify.go builds https://<node>/contact/...), app/sdk, app/client/web (how the web app derives the server node/host and builds invite links /#/access).
Deliver (keep each change small and rebaseable; Go tests + web build green):
1. Loopback bind: accept `-p 127.0.0.1:7100` (host:port) as well as `-p 7100`; add a test.
2. Base path: the web app must work when served under a sub-path. Build with a relative base (vite base './' or equivalent) so one build works at any mount; derive the API node as location.host + the mount path (the path the app was loaded from, e.g. from document base / window.location.pathname up to the app root) so SDK requests go to https://<host>/<prefix>/account etc.; invite/access links must keep the prefix. The node's federation `domain` config value may then contain a path ("sat.example/matrix/h/host/app/klatch"); make sure the server builds federation URLs as https://<domain>/contact/... without breaking on the path (domain string concatenation) and that any validation of `domain` allows a path. Add unit tests where code paths are pure.
3. Go-native image transforms: when ImageMagick `convert` is unavailable, images must still display. Implement the image transforms (icopy, ithumb, ilg — see net/transform/*.sh and net/server/internal/transcodeUtil.go) in Go (image/jpeg, image/png, image/gif decode; golang.org/x/image/draw for resize — BSD-3) as the default for images, keeping script-based audio/video untouched (they stay disabled in the MVP). Add tests with small fixture images.
4. Local proof on this machine (Dragon: short runs only): build the server (CGO_ENABLED=1 go build; macOS clang is fine), run it on 127.0.0.1:7100 with a temp data dir, and simulate the prefix with a tiny Go or node reverse proxy that strips /matrix/h/test/app/klatch and sets X-Forwarded-Prefix; with curl verify: admin bootstrap, account create/login, a group + text + an image upload whose ithumb/icopy assets become ready (no `convert` on PATH), and that GET <prefix>/ serves index.html whose asset URLs resolve under the prefix. Stop the server and delete temp data.
Verification: `cd net/server && go test ./...` green; `cd app/client/web && yarn --frozen-lockfile && yarn test && yarn type-check && yarn build` green; local proof transcript (commands + key outputs) in the commit message. Remove node_modules/dist before finishing.

== RULES (all S518 workers) ==
- Your cwd is your isolated workspace (a full git clone). Stay inside it; relative paths only; never follow git remotes. Do not push. Do not switch branches. git add files BY NAME. Keep the bashy-injected BASHY_AGENT and the commit trailers weave gives you.
- Plan of record: umbrella docs/sprint-518-master-execution-plan.md (Klatch = our rebranded fork of Databag, github.com/qiangli/klatch, pinned to upstream main 45343338582b394f4f5fdb4b7e271ea1dc8dd5fd; deadline Sat 2026-10-31 PDT).
- Dragon (this machine) is for editing + short unit tests only: no podman/docker, no full builds of large trees, keep disk use small (delete node_modules you created before finishing).
- Public repos (klatch fork, bashy, yoke): NEVER write real hostnames, IPs, user names or emails into files or commit messages. Hosts come from env/config only.
- Remote Linux access when your story needs it: read `bashy claim list | grep s518` — the host:alien:s518-klatch intent carries the ACCESS line (outpost ssh ...). Use only /opt/s518-klatch on that host, loopback ports 7100/7101 (7000 is taken). Never touch cloudbox, DKS, k3s or the 80/443 front there.
- Credit: Klatch is based on Databag by Roland Osborne (github.com/balzack/databag), Apache-2.0. Keep upstream LICENSE and copyright headers.
- Done = the story's verification passes and is committed. State EXACT measured results in the commit message. If stuck after 3 serious attempts, commit what works plus a BLOCKERS.md with diagnosis, then exit 0. Headless: exit cleanly (exit 0) when done. COMMIT YOUR WORK.
