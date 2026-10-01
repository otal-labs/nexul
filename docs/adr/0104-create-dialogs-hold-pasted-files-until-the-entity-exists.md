# Create dialogs hold pasted files until the entity exists

An attachment is owned by exactly one existing doc, ticket, conversation, or memory (ADR 0027), so a New ticket or
New doc dialog has nothing to attach a pasted screenshot to. ADR 0050 answered that by keeping the ticket dialog a
plain textarea, and the doc dialog's editor simply ignored pasted files. Both read as broken: the same paste works on
the page the moment the entity exists.

Decision: a create dialog's editor holds pasted, dropped, or picked files in the browser and shows each from a local
object URL. On Create the entity is created with those files left out of its body, each file is uploaded against the
new entity, and the body is saved once more with the stored paths. A file whose upload fails is named in a toast and
dropped from the body; the entity and every other file stay. Closing the dialog, or Create more, releases the held
files. The ticket dialog now uses the same editor as the doc dialog.

The trade-off: a create with files is two writes, so the entity briefly exists without them, a doc gains a second
version, and a failed second save leaves the entity without its files (the save's own toast says so). No object URL
is ever stored, so a half-finished create shows missing images, never broken ones.

Rejected: an ownerless upload claimed by the entity later, which needs server-side orphan cleanup and an attachment
that belongs to nobody, against ADR 0027's single-owner rule.

Supersedes the create-ticket textarea paragraph of ADR 0050. Decided 2026-10-01.
