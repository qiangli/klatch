---
id: 4491f0625016
kind: feature
title: Klatch participant protocol and crypto helper
seq: 2828
status: done
priority: p0
created: 2026-11-08T17:43:16.529728Z
weave: 14
assignee: codex-gpt6-astra
sprint: 597
sprint_id: ce9248d4-2579-5c97-857e-de17d182395f
sprint_title: Klatch participant CLI MVP
closed: 2026-11-08T18:49:00.636549Z
closed_by: codex-gpt6-sol-klatch
---

Implement minimal Node TypeScript helper in Klatch for protocol, existing app sessions/MFA, client seal crypto, files, and SDK sync watch. Support contact/conversation/member/message/attachment operations via JSON input/output with stable IDs; no text/secrets in argv; prefix assistant-authored outgoing text [Assistant: NAME]. Bash# bashy/examples/klatch/klatch.bsh is the canonical user command and shaping entrypoint; Node helper is internal, small, and reusable. No new auth, automatic replies, daemon, durable replay, or retry of uncertain sends. ALL builds/tests on the claimed shared test host through the established outpost SSH route; development host editing/coordination only. Freeze helper contract with Bash# owner.
