# 06 — Send a turn's images on protocol 2

**What to build:** Before `message.dispatch`, `t3clientv2` sends `assets.persistChatAttachments` with
the same thread id and message id, each image as `data:<lowercased mime>;base64,…`, and puts the
returned references verbatim into `message.dispatch.attachments`. Only gif, jpeg, png and webp;
anything else is skipped with a `slog` Warn and an `ActivityNote`. A turn with no images makes no
persist call and sends `attachments: []`.

**Blocked by:** 05

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] Persist failure → no dispatch, StartTurn returns the error
- [ ] Returned ids used verbatim; unsupported MIME skipped and noted; no images → no persist call
