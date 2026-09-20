# 12 — What shape does T3 Code accept for turn attachments?

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Ticket 11 found that the T3 Code wire message `thread.turn.start` carries an
`attachments` array which Nexul's client always sends empty
(`internal/t3client/thread.go:104`), and that whether T3's server reads it,
and in what shape, is only answerable from T3 Code itself. Establish from
T3 Code's source (the GitHub project, or the local install on this machine
under `~/.t3` or wherever the desktop app keeps its server code):

- The server-side handler for `thread.turn.start`: does it read
  `attachments`, and what is each element's shape (inline bytes, a file
  path, a URL, a MIME type, a name)? Size or type limits?
- How the T3 desktop app itself attaches an image from a phone or the
  composer: what it sends on the same message, as the ground truth of the
  shape.
- Whether a URL that requires Nexul's authentication (the
  `/api/attachments/{id}` gateway) could be fetched by T3, or whether bytes
  must be inlined.
- The T3 Code release tag Nexul pins (search `internal/t3client` for a
  version or tag) so the answer is checked against that version.

Findings go to `.scratch/plays/research/12-t3-turn-attachment-shape.md`.

## Answer

Shape known, confirmed against T3 Code's public source (HEAD
`32e8b2584556c0c55ce0d1d8f72b506cb771d60d`) and the installed 0.0.43-nightly
desktop build. A new image attachment is `{type:"image", name, mimeType,
sizeBytes, dataUrl}` — bytes inlined as base64, capped at 10 MiB / 14M chars.
No variant has a `url` field, and the server never fetches attachments
remotely, so a Nexul-authenticated URL cannot work: bytes must be inlined.
Nexul's pinned 0.0.34 wasn't independently verifiable (tag pruned from the
repo), but the field predates it per ticket 11.

Research: `.scratch/plays/research/12-t3-turn-attachment-shape.md`
