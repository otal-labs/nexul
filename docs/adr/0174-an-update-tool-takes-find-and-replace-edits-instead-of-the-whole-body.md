# An update tool takes find-and-replace edits instead of the whole body

To change one sentence, an agent had to resend a whole body to `doc_update`, `ticket_update`, `play_update` or
`memory_update`. A long body made that slow and costly, and every resend risked dropping a line the agent had not
meant to touch.

Decision: each of those four tools takes `edits`, a list of `{find, replace}` pairs applied in order to the body the
matching get tool returns (a doc's, a ticket's description, a play's instructions, a memory's body).

- **Exactly one match, all or nothing.** Each `find` must occur once in the text the earlier edits left. An empty
  find, no match, or several matches fails the call with the edit's position and how to fix it, and nothing is
  saved.
- **Not with the whole text.** `edits` is refused beside the body field and beside `revert_to_version`. It combines
  with the other patch fields of the call, such as a ticket's status.
- **One ordinary save.** The result goes through the same update as a whole body: one version (a save that changes
  nothing still writes none), the same `via` rules, events, permissions and doc lock, and the live-room reset of
  ADR 0124. Each domain's use-case exposes `ApplyEdits`, which reads the body as its get tool renders it, so a
  `find` copied from a get always matches.
- **The rule is said once.** The server instructions carry the edit rules; each tool's `edits` description is a few
  words (ADR 0171). The shared logic is `internal/platform/textedit`. The instructions sit at their 2,048-character
  cap, so the rule's room came from shorter wording and dropping "Work started here is recorded as started via
  MCP", which the record itself shows.

The cost: `advertisedBudget` rises from 185,000 to 186,500 characters. `tools/list` grew from 184,974 to 186,172,
about 300 characters of schema per tool, paid in every session. One call with a few edits replaces resending a
whole body, which saves far more on each use than the schema costs per session.

The trade-off: the text is read and saved in two steps, so a body saved between them is overwritten, as with a
whole body but over a shorter window.

Rejected: a patch format such as a line range or a diff, which breaks when the text shifts; replace-all and
first-match semantics, which edit the wrong place silently; HTTP support, which no browser flow needs.

Decided 2026-10-30.
