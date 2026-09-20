# 11 — Can a harness turn carry images?

Researched 2026-09-16, on master. Answers whether a play/turn can hand T3
Code an embedded image instead of a "[image omitted]" placeholder.

## 1. `harness.Client.StartTurn` / `TurnPrompts` carry text only

`harness.TurnPrompts` (`internal/harness/harness.go:108-112`) is two strings:

```go
type TurnPrompts struct {
	Full        string
	Incremental string
}
```

`Client.StartTurn` (harness.go:132) takes `(ctx, Target, title string, prompts
TurnPrompts)` — no attachment-shaped parameter anywhere in the interface.
Every other harness-neutral type (`Session`, `Target`, `Update`, `Snapshot`,
`Activity`, `Approval`) is also plain text/metadata, no byte-carrying field.

The text is plain-text already by the time it reaches the interface: the
whole path from a chat message to the wire is text-only —
`internal/agent/pipeline.go:281` builds each `ContextMessage` as
`{Author: ..., Body: m.Body, At: m.CreatedAt}`, dropping `chat.Message`'s own
`AttachmentID` (`internal/chat/model.go:81-82`, "a nullable attachments door;
no v1 use-case sets it yet") on the floor. `ComposePrompt`/
`ComposeIncrementalPrompt` (`internal/agent/prompt.go`) only ever concatenate
strings, and `buildTurnPrompts` (pipeline.go:274-291) hands the result
straight into `harness.TurnPrompts{Full: ..., Incremental: ...}`. So even
before the harness seam, nothing upstream threads an attachment through.

## 2. T3 client: attachments field exists on the wire, unused

`internal/t3client` speaks T3's internal Effect RPC protocol over one
WebSocket, pinned to v0.0.34 (client.go:1, `PinnedT3Version` client.go:27).
No vendored OpenAPI/proto; the wire shapes are reverse-engineered
(thread.go:292: "wire shapes below decode leniently... since the protocol is
reverse-engineered").

The `thread.turn.start` command's message payload already has an
`attachments` field:

```go
// thread.go:100-105
type turnMessage struct {
	MessageID   string `json:"messageId"`
	Role        string `json:"role"`
	Text        string `json:"text"`
	Attachments []any  `json:"attachments"`
}
```

But `Client.StartTurn` (thread.go:164-179) always sends it hardcoded empty:
`Attachments: []any{}` (thread.go:173), asserted in
`client_test.go:90` (`assert.Equal(t, []any{}, msg["attachments"])`). No
comment records what shape T3 expects inside that slice, no size/type limit
is documented anywhere in this repo, and nothing in `internal/t3client`
reads or round-trips an attachment back off the wire (no `Attachments` field
is decoded from `thread.message-sent`/`streamItemUpdate`, thread.go:294-414).

**Verdict for this repo: unknown whether T3 Code's server actually accepts
non-empty `attachments`.** The field's presence and `any` element type are
the only evidence the wire protocol has a slot for it — this client has
never populated it, so nothing proves the server reads it, what shape an
element needs (inline bytes? a file reference? a data URL?), or what
size/MIME limits apply. That answer, if it exists, lives in the T3 Code
project (its `thread.turn.start` handler, wherever T3 documents its own
Effect RPC schema), not in this socket's client side.
`h.subscribeAndStart` (`internal/t3client/harness.go:140-150`), the only
caller of `Client.StartTurn`, never sets `Attachments` either.

## 3. Docs export and attachment serving

`richtext.JSONToMarkdown` (`internal/docs/richtext/to_markdown.go:9-22`)
walks the canonical Tiptap doc and dispatches an `"image"` block node to
`renderImage` (to_markdown.go:39-40, 110-118):

```go
func renderImage(n Node) string {
	src, _ := n.Attrs["src"].(string)
	alt, _ := n.Attrs["alt"].(string)
	title, _ := n.Attrs["title"].(string)
	...
	return "![" + escapeMarkdownText(alt) + "](" + src + ")"
}
```

`src` is already the attachment's serving URL — ADR 0027 states the export
"already renders each file as `![name](/api/attachments/<id>)`"
(`docs/adr/0027-attachment-bytes-live-in-sqlite.md:22`) — so a doc's embedded
image becomes a standard markdown image reference pointing at the app's own
API, not an inlined data URL. `Service.ExportMarkdown`
(`internal/docs/usecase.go:265-271`) and the MCP `docToMarkdown` helper
(`internal/docs/mcp.go:143-146`) both call the same `richtext.ToMarkdown`.

Attachment bytes live in SQLite, capped at 10 MB (ADR 0027), served by
`GET /api/attachments/{id}` (`internal/attachments/handler.go:31`, routed at
`server/cmd/routes.go:76`) → `h.serve` (handler.go:89-106): sets
`Content-Type` from the sniffed type, `Content-Disposition: inline` only for
raster images (`a.Inline()`, `model.go:35`) else `attachment`, always
`X-Content-Type-Options: nosniff`. Auth is the normal bearer-gated gateway —
a plain `<img src>` can't use it (ADR 0027), the browser fetches bytes
through the API client and renders an object URL. Attachments deliberately
have no MCP tools (ADR 0027) and, per §1, no chat/turn use-case sets
`chat.Message.AttachmentID` yet — the door exists on the model, unopened.

## 4. Smallest harness-neutral seam (types only, no behavior)

Per ADR 0054, only harness-neutral types may cross `harness.Client`; T3
specifics (Effect RPC envelopes, the `attachments: []any` wire shape) stay
inside `internal/t3client`.

```go
// internal/harness/harness.go
type Attachment struct {
	Name string
	MIME string
	URL  string // e.g. "/api/attachments/<id>" — reuses the existing gateway, no new byte-carrying path
}

type TurnPrompts struct {
	Full        string
	Incremental string
	Attachments []Attachment // empty today; nil-safe zero value
}
```

`URL` over inline `Bytes` follows ADR 0027's own reasoning (10 MB cap,
already-authenticated gateway, no duplicating bytes over the tool protocol)
and matches how docs export already links images. Unresolved by this repo:
what `internal/t3client.Harness.StartTurn` (harness.go:92-129) should put in
`turnMessage.Attachments` (thread.go:104) — the expected element shape is
exactly §2's unknown. Guessing it and shipping unverified risks either a
silent no-op (T3 may ignore unrecognized fields, per the "decode leniently"
posture) or a fatal `Defect` (client.go:278-281). If T3 needs bytes/base64
rather than a URL it can dereference itself, this server would also need to
fetch the attachment server-side before dispatching — T3 runs on the user's
own machine, not here.

## Bottom line

Cannot confirm from this repo alone. Text-only is proven end-to-end
(`internal/harness/harness.go:108-112`, `internal/agent/pipeline.go:281`,
`internal/t3client/thread.go:173`). Whether T3 Code's server would accept
attachments if this repo sent them is unknown here — the wire type has a
slot (`thread.go:104`) that's never been exercised, and the answer lives in
the T3 Code project, not in `internal/t3client`.
