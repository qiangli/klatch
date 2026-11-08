# Participant protocol helper

Bash# `klatch.bsh` is the primary participant command and shaping entrypoint.
This internal Node TypeScript helper supplies the existing app-session HTTP
protocol, seal crypto, files and SDK Connection watch adapter. It adds no server
API, bot identity, daemon, durable event replay or mutation retry.

Requires Node 22+. Install dependencies with `npm ci` and build with
`npm run build` in this directory. Run `npm test` for the focused gate.

## Frozen internal command contract

```
node app/client/cli/dist/main.js COMMAND [--input FILE|-] [--session FILE] [--json]
```

Input is one JSON object, from stdin through EOF by default, or a UTF-8 file.
No positional payloads, inline secret flags, environment credential fallback,
or interactive prompts. `--json` is accepted; output is always JSON.
The maximum input is 1 MiB. Input files must be private regular files owned by
the caller (0600 on POSIX); symlinks are refused. Session defaults to
`~/.config/klatch/session.json`; use separate session files for separate accounts.
Only the app token, server and account GUID persist, with atomic 0600 writes.
On Windows the caller must restrict the containing directory ACL to their user.

Success: stdout `{ "ok": true, "result": ... }` followed by newline, exit 0.
Failure: stderr `{ "ok": false, "error": { "code": "..." } }`, exit 1
(or 2 for usage/invalid JSON/input). No raw transport exceptions, URLs, passwords,
MFA codes or tokens are logged. Returned credential-named fields are removed.
Earlier draft announcements using `data` as the envelope key are superseded.

IDs are opaque server IDs, never display names or list indexes. `cardId` on
content commands selects a connected contact's remote owner scope; omit for
owned content. `channelId`, `topicId`, `assetId` retain their server meanings.
The result retains API object shapes and IDs; list results are arrays.

| Command | JSON input fields (question mark means optional) |
| --- | --- |
| auth-login | server, username, password, code? (mfa alias accepted) |
| auth-status / auth-logout | {} |
| seal-unlock | sealPassword |
| contacts-list | {} |
| contacts-invite | guid, server? |
| contacts-accept | cardId |
| conversations-list | cardId?, sealPassword? |
| conversations-create | title, cards? (array of card IDs), sealed? (default true), sealPassword? |
| conversations-show | channelId, cardId?, sealPassword? |
| conversations-rename | channelId, title, sealPassword? (owner only) |
| conversations-leave | channelId, cardId, sealPassword? (contact scope only) |
| members-list | channelId, cardId?, sealPassword? |
| members-add / members-remove | channelId, memberCardId, sealPassword? (owner only) |
| messages-list | channelId, cardId?, sealPassword?, count?, begin?, end?, revision? |
| messages-show / messages-delete | channelId, topicId, cardId?, sealPassword? |
| messages-send | channelId, text, assistant, cardId?, sealPassword? |
| messages-edit | channelId, topicId, text, assistant, cardId?, sealPassword? |
| attachments-upload | channelId, topicId, path, text, assistant, cardId?, sealPassword? |
| attachments-download | channelId, topicId, assetId, path, cardId?, sealPassword? |
| watch | channelId?, cardId?, sealPassword?, count?, begin?, end?, revision? |

Server URLs may include an app mount path, for example
`https://example.test/proxy/app`, with or without a trailing slash. Login, API
requests, remote contact servers and watch retain that prefix. Credentials, query
strings and fragments are forbidden in server URLs.

Login uses `/account/apps`, including its existing MFA challenge behavior.
On `MFA_REQUIRED_OR_INVALID`, explicitly invoke login again with a code supplied
through private JSON. Logout revokes the app session before removing its file.
`seal-unlock` validates the account seal; it does not cache decrypted keys.
Supply `sealPassword` anew for operations on sealed conversations. Missing or
incorrect keys fail closed. Creation defaults to sealed; plain creation requires
`sealed:false`. Removing a member does not revoke keys or history already held.

Every message send/edit and attachment caption requires `assistant` and `text`.
The stored text is `[Assistant: NAME] TEXT`, including when editing an existing
message. Names allow letters, digits, spaces, underscores, periods and hyphens
(up to 64 characters). Edit/delete/upload require the session account's own topic.
An upload appends a file to an existing topic and replaces its caption with the
attributed text. Files are limited to 16 MiB; downloads refuse to overwrite files.
Sealed attachment blocks use the existing SDK encryption representation.

Watch is foreground NDJSON: each line has the success envelope with a result
`{event:"status",status}` or `{event:"snapshot",result}`. Snapshots are current
conversation or message lists, not exactly-once events. SDK reconnect triggers
resynchronization; duplicates and gaps outside the requested window are possible.
SIGINT/SIGTERM closes the connection. No background process is started.

No HTTP request is automatically retried. A transport error after a write is an
uncertain outcome: inspect server state before explicitly invoking another write.
Contact invitation and attachment upload have multiple writes; a failure can leave
a pending contact or uploaded block. Membership updates likewise are not atomic.
Do not blindly retry. Foreground socket reconnect is the sole automatic reconnect.

## Verification evidence

The focused gate (`npm test`) compiles TypeScript, bundles the existing SDK
Connection and RSA dependency, then runs eleven Node tests. The remote Node 22.15.0
run passed 11/11 with zero failures or skips. Coverage includes app login/MFA
payloads, private session permissions and symlink refusal, mandatory attribution,
non-retried transport failure, author ownership, attachment byte round-trip and
no-overwrite behavior, SDK socket reconnect/resync/abort, seal unlock, RSA channel
key recovery and encrypted topic decoding. Proxy-prefix regressions cover login,
API requests, remote contact scope, and socket reconnect/resync. HTTP routes use fixtures; watch uses
a real loopback WebSocket server. This is not evidence of live-account end-to-end
or Bash# wrapper acceptance. Dependency install audit reported zero vulnerabilities.
