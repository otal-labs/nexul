# 11 — Can a harness turn carry images?

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

The owner sends images to T3 Code from a phone, so a play run should be
able to hand the harness the images embedded in an inlined memory rather
than a "[image omitted]" placeholder. Establish, with file and line
references:

- What `harness.Client.StartTurn` and `harness.TurnPrompts`
  (`internal/harness/harness.go`) can carry today: text only, or
  attachments.
- What the T3 Code client in `internal/t3client` sends when it starts a
  turn, and whether T3 Code's server API for a turn accepts attachments or
  images (its request shape, any size or type limits). Look at the T3 Code
  client code and any vendored or documented API it targets; if the API is
  only discoverable from the T3 Code repo or docs, say where and what was
  found.
- How the docs domain exports a doc to markdown today and how an embedded
  image is represented in that export (`internal/docs`, attachments), so
  the export can either emit an attachment reference or the placeholder.
- The smallest change to the harness interface that would let a turn carry
  attachments without T3 leaking into the seam (ADR 0054).

Findings go to `.scratch/plays/research/11-harness-image-attachments.md`.

## Answer

Unknown from this repo, not "cannot": `harness.TurnPrompts` and the whole
prompt path are text-only today (`internal/agent/pipeline.go:281` drops
chat's own `AttachmentID`). T3's `thread.turn.start` wire message already
has an `attachments: []any` field (`internal/t3client/thread.go:104`), but
this client always sends it empty and never decodes one back — whether T3's
server reads it, and its expected shape, lives in the T3 Code project, not
here. The `"[image omitted]"` placeholder stays correct until that's
verified upstream. Smallest seam if T3 turns out to accept them: add
`TurnPrompts.Attachments []Attachment{Name, MIME, URL}` pointing at the
existing `/api/attachments/{id}` gateway, T3 specifics staying inside
`internal/t3client` (ADR 0054). Details:
`.scratch/plays/research/11-harness-image-attachments.md`
