# 06 — Send a turn's images on protocol 2

**What to build:** Before `message.dispatch`, `t3clientv2` sends `assets.persistChatAttachments` with
the same thread id and message id, each image as `data:<lowercased mime>;base64,…`, and puts the
returned references verbatim into `message.dispatch.attachments`. Only gif, jpeg, png and webp;
anything else is skipped with a `slog` Warn and an `ActivityNote`. A turn with no images makes no
persist call and sends `attachments: []`.

**Blocked by:** 05

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [x] Persist failure → no dispatch, StartTurn returns the error
- [x] Returned ids used verbatim; unsupported MIME skipped and noted; no images → no persist call

## Comments

- **Where it lives.** `internal/t3clientv2/images.go` is `turn.persistImages`; `turn.dispatch` calls it before
  `thread.runtime-mode.set` and `message.dispatch`, so a refused upload leaves the thread untouched. ADR 0114 gained an
  "images" bullet.
- **Full prompt only.** Images ride with the Full prompt (a new thread, a recreated one, or an import with no completed
  run), as on protocol 1. An Incremental prompt uploads nothing, whatever `TurnPrompts.Attachments` holds.
- **One note, not one per image.** Unsupported images are logged at Warn one by one and noted once, listing their names
  ("Not sent to T3 Code: a.svg, b.bmp. It takes only gif, jpeg, png and webp images."). StartTurn's update buffer is 16,
  and queueing a note per image before the pump starts could block it.
- **One upload call.** T3 derives each id from the message id and the image's index in the call, so splitting a turn's
  images over several calls would give two images the same id. The call carries every supported image.
- **Lowercased mime in both places.** T3 compares the data URL's mime with `mimeType` lowercased, so `mimeType` goes out
  lowercased as well as the data URL's.
- **Not checked.** T3's own limits (10 MiB per image, 100 attachments, 80 MiB per message) are not pre-checked; the
  attachments domain already caps one image and a turn's total below them, and a breach would come back as the upload's
  own error. A response with fewer references than images is not checked either: T3 answers one per image. The images
  travel in one frame, up to 25 MiB raw (a turn's total cap), the same exposure protocol 1's inline attachments had.
- **`t3rpctest`.** The fake answers `assets.persistChatAttachments` with one reference per image (id
  `<threadId>-ref-<index>`) and records the payload on `Persisted`; `CommandCauses["assets.persistChatAttachments"]`
  fails it. It does not validate the payload, so tests assert the exact payload.
- **Ticket 16.** The walkthrough should send a real image on both a new and an imported thread and confirm the agent
  describes it; nothing here was checked against a running nightly.
