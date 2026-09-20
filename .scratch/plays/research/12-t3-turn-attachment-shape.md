# 12 — What shape does T3 Code accept for turn attachments?

Researched 2026-09-16, on master. Resolves ticket 11's open question: whether
T3 Code's server accepts a non-empty `attachments` array on `thread.turn.start`
and, if so, in what shape.

## 1. Where the answer came from

Two independent sources, cross-checked and byte-for-byte consistent:

- The locally installed T3 Code desktop build,
  `~/Applications/T3Code/T3-Code-0.0.43-nightly.20260916.1811-x86_64.AppImage`.
  Extracted with `--appimage-extract`, then unpacked
  `resources/app.asar` with `@electron/asar`. Both
  `apps/desktop/dist-electron/main.cjs` (minified, defines the wire schemas)
  and `apps/server/dist/bin.mjs` (minified, the actual dispatch handler) carry
  the same Effect Schema definitions.
- The public GitHub source, `pingdotgg/t3code`, HEAD commit
  `32e8b2584556c0c55ce0d1d8f72b506cb771d60d`,
  `packages/contracts/src/orchestration.ts`. This is the readable,
  unminified original the two bundles above were built from — every field
  name, constant, and comment matches.

`~/.t3/worktrees/nexul/t3code-b13ad8cf` was a dead end: despite the name, it's
a worktree of Nexul's own repo (`module github.com/otal-labs/nexul`) that T3
Code itself created for an agent branch, not T3 Code's source.

## 2. `thread.turn.start` attachment shape (`orchestration.ts`)

The wire command is `ClientThreadTurnStartCommand` (what a client sends over
`orchestration.dispatchCommand`):

```ts
attachments: Schema.Array(Schema.Union([UploadChatAttachment, ChatAttachment]))
```

Two families, discriminated by whether the element carries bytes:

**`UploadChatAttachment`** — a brand-new attachment, `Union([UploadChatImageAttachment])`
(images only; there is no "upload a new file" variant today):

```ts
{
  type: "image",
  id?: ChatAttachmentId,       // optional client-side id, max 128 chars, /^[a-z0-9_-]+$/i
  name: string,                // max 255 chars
  mimeType: string,            // max 100 chars, must match /^image\//i
  sizeBytes: number,           // <= 10,485,760 (10 MiB)
  dataUrl: string,             // base64 data: URL, max 14,000,000 chars
  source?: SnapShotSource,     // present only for screenshot-tool attachments
}
```

**`ChatAttachment`** — a reference to bytes the server already has, no
`dataUrl` field at all: `Union([ChatImageAttachment, ChatFileAttachment, ChatUnknownAttachment])`,
each `{ type, id, name, mimeType, sizeBytes, source? }`. `ChatUnknownAttachment`
deliberately excludes `type: "image" | "file"` so unrecognized future
attachment kinds still decode.

Supported image MIME types: `image/gif`, `image/jpeg`, `image/png`,
`image/webp`. Size caps: `PROVIDER_SEND_TURN_MAX_IMAGE_BYTES = 10485760`,
`PROVIDER_SEND_TURN_MAX_FILE_BYTES = 52428800`,
`PROVIDER_SEND_TURN_MAX_IMAGE_DATA_URL_CHARS = 14000000`,
`PROVIDER_SEND_TURN_MAX_INPUT_CHARS = 120000` (all in
`packages/contracts/src/orchestration.ts`).

## 3. Server-side handler proves it's read, not just declared

`apps/server/dist/bin.mjs` (dispatch-command path) does, for each attachment
in the array:

```js
if (!("dataUrl" in attachment)) {
  // ChatAttachment case: must already be claimed on this server's local disk
  const claim = planAttachmentClaim({ attachmentsDir: serverConfig.attachmentsDir, threadId, attachmentId: attachment.id });
  ...
}
const parsed = parseBase64DataUrl(attachment.dataUrl);
if (!parsed || !parsed.mimeType.startsWith("image/")) return /* OrchestrationDispatchCommandError */;
const bytes = Buffer.from(parsed.base64, "base64");
if (bytes.byteLength === 0 || bytes.byteLength > 10485760) return /* error */;
// ...writes bytes to serverConfig.attachmentsDir on the local filesystem
```

This is the ground truth for how the desktop composer attaches an image: it
sends `UploadChatImageAttachment` with the bytes base64-encoded inline as
`dataUrl`; the server decodes and persists them to its own local
`attachmentsDir`, then answers future turns in the same thread with the
plain `ChatAttachment` (id-only) form. There is no network fetch of the
attachment anywhere in this path — the server never dereferences a remote
URL for an attachment, new or referenced.

## 4. Can a Nexul-authenticated URL work, or must bytes be inlined?

**Bytes must be inlined.** No attachment variant — `UploadChatImageAttachment`,
`ChatImageAttachment`, `ChatFileAttachment`, `ChatUnknownAttachment` — has a
`url` field. The only way to hand the server new image bytes is
`dataUrl` (a base64 `data:` URL), capped at 10 MiB raw / 14M base64 chars.
Per ADR 0029, T3 Code runs on the mentioning user's own paired environment,
not Nexul's — so Nexul's server can't expect T3 to reach back into
`/api/attachments/{id}` even if a URL field existed; T3 has no code path
that would fetch it. Nexul would have to fetch the attachment bytes itself
(server-side, through its own gateway) and inline them as a `dataUrl` in the
`thread.turn.start` message before sending it over `internal/t3client`'s
WebSocket.

## 5. Version checked vs. Nexul's pin

Nexul's client pins `PinnedT3Version = "0.0.34"`
(`internal/t3client/client.go:27`); ADR 0029 records the pinning policy
("pins a stable release tag and only warns... when a paired server's base
release has moved") without naming the number.

This research checked commit `32e8b2584556c0c55ce0d1d8f72b506cb771d60d`
(current GitHub HEAD) and the locally installed `0.0.43-nightly.20260916.1811`
build — both newer than 0.0.34. **Not found:** an exact `v0.0.34` tag —
`mcp__github__get_tag` 404s on it, and `list_tags` only shows recent
0.0.41/0.0.42/0.0.43 nightlies/previews, so the tag was pruned. Cannot
confirm byte-for-byte that the schema was identical at 0.0.34; the
`attachments: []any` field already existed then (ticket 11), and the shape
above is unlikely to have changed structurally, but the exact size caps and
field names at that specific pin are unverified.

## 6. Bottom line

Shape is known and confirmed against two independent decompiled/source
copies: a new image attachment is `{type:"image", name, mimeType, sizeBytes,
dataUrl}` with inline base64 bytes; a URL to Nexul's gateway cannot work —
T3 never fetches attachment bytes remotely, whether new or on a fresh
pairing. Any Nexul integration must inline bytes it has already fetched.
