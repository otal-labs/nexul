# 05: Redrafting after a source is added or changes

Type: grilling
Status: resolved
Blocked by: 03

## Question

A new source, or a change to one, redrafts only the answers it touches.
Decide:

- How a change is noticed: a person pressing redraft, or the page marking
  sources changed since the last drafting (a doc or memory version, a
  commit in the checkout).
- How the run decides which answers a source touches, and what a redraft
  of an already confirmed answer looks like: a suggested change the
  person accepts or dismisses, never an overwrite.
- Whether a redraft leads to regenerating the memory the same way a
  changed answer does today.

## Answer

- **Noticing a change** is a signal on the page, never a trigger. The
  "Draft answers" button gets the warning dot, the same one Regenerate
  uses, when a source was added after the last finished drafting run, or
  when a doc, memory, or pasted text source changed after it (their
  `updated_at` against the trail's start). Paths and whole projects get no
  marker: the server cannot see a checkout move, so after a commit to the
  standards the person presses the button themselves. Nothing redrafts on
  its own, for the same reason drafting never starts on its own.
- **Which answers a source touches**: the run does not try to work that
  out from the change. Each drafting run reads every follow source and
  drafts every template question, which costs reading time but cannot
  miss a question an edit quietly affected. What comes back to the person
  is filtered, not the reading:
  - An unanswered question gets its draft, replacing any earlier one.
  - An answered question gets a draft only where the sources now say
    something different from the stored answer. The instructions tell the
    run to leave agreeing questions alone, and the server also drops a
    draft whose picks and text equal the stored answer.
  - A skipped question is treated like an unanswered one: a source may now
    speak to it.
- **A draft on an answered question is a suggested change.** The card
  shows the confirmed answer with the suggestion under it and where it
  came from. Accept saves the suggestion as the answer; Dismiss deletes
  the draft; the answer is never overwritten. A dismissed suggestion comes
  back only if a later run drafts it again, which happens only when a
  source still disagrees: the person can then change the source, or
  accept.
- **Regenerating the memory** works as today: accepting a suggestion is a
  saved answer, so the memory column shows its warning dot and
  "Regenerate the memory". A redraft alone changes nothing.
