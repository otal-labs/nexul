# 05: What Clarify via AI asks

Type: grilling
Status: resolved
Blocked by: 04

## Question

The play's instructions, the content the owner decides:

- What counts as a gap in a client's doc, and what the agent should never
  ask a client (implementation detail, things the developer decides).
- How many questions in a round, and in what order.
- Whether each question offers the client a recommended answer, and how
  options are worded for someone non-technical.
- How the agent uses the developer's instructions for the run and the
  client's "Anything else?".
- When the agent calls "no gaps left".
- Raised by the research: whether the client sees an option marked
  "(Recommended)" and pre-picked, as the interview does; whether the
  first-person "Why I'm asking:" line suits a client; and the inbox wording
  of the two new notifications.

## Answer

Decided with the owner.

- **What it asks**: what the authors need, in their own terms.
  - Functional: who uses it and what each of them does, the steps of each
    flow, business rules and their exceptions, the data they have or must
    keep, what is in and out of scope, what finished looks like.
  - Non-functional: how many people use it and when, how fast it must
    feel, when it must be available, who may see what, devices and
    languages, accessibility, systems it must work with, legal or industry
    rules, deadlines, budget, what matters most.
  - Never how to build it (hosting, databases, frameworks, architecture):
    anything technical left unclear goes to the developer in the run's
    reply.
- **A round**: three to six questions, biggest unknowns first, grouped by
  the doc's sections.
- **Suggested answers**: the option most projects pick is listed first and
  labelled "(Suggested)", but never pre-picked; the client chooses it
  themselves.
- **The why-line** reads "Why we're asking:", in the team's voice.
- **No gaps left** when the developer could turn the doc into tickets
  without asking the authors anything more. A skipped question that still
  matters is asked once more in a later round; if it is skipped again, the
  write lists it under a short "Open points" section and the reply says so.
  The developer's instructions for a run win over all of this.
- **Inbox wording**: to the doc's watchers, "New questions on <doc title>";
  to the round's starter, "Questions answered on <doc title>" with a
  summary such as "Round 2 · 5 answered, 1 skipped". Neither mentions AI.
- Tightened after the walkthrough: options rules, the check before no
  gaps, and no invented detail in the rewrite.

### The play's instructions (code default and instance template)

> Clarify this doc for the people who wrote it. Read it with `doc_get`: its
> body and its clarification, every earlier round with its questions,
> answers, skipped questions, and "Anything else?" text. The instructions
> for this run, if any, win over everything below.
>
> Find the gaps in what the doc says its authors need. Ask about what it
> must do: who uses it and what each of them does, the steps of each flow,
> the business rules and their exceptions, the data they have or must keep,
> what is in and out of scope, and what finished looks like. Ask about how
> well it must do it, in their terms: how many people use it and when, how
> fast it must feel, when it must be available, who may see what, devices
> and languages, accessibility, the systems it must work with, legal or
> industry rules, deadlines, budget, and what matters most. Never ask how to
> build it (hosting, databases, frameworks, architecture); those are the
> developer's, so put anything technical left unclear in your reply.
>
> Do not ask what an earlier round answered, and do not repeat a question
> still waiting for an answer. Ask a skipped question once more only if it
> still matters.
>
> If gaps remain, post one round with `doc_update` `questions`: three to
> six, the biggest unknowns first, grouped by the doc's sections. Write each
> in plain words for someone non-technical, with two to four concrete
> options that each stand on their own: the one most projects pick first and
> labelled "(Suggested)", no other option labelled, multi-select only where
> the options can be combined, and a one-line why that says why it matters
> to them. If the last round has "Anything else?" text, send
> `anything_else_reply` with it: one plain line that answers it, or says
> which of this round's questions follow it up; never name a round number in
> it. Never use your question tool: post the round and end your turn.
>
> Before you call it done, check that every area above was either asked
> about or is already answered in the doc; anything you set aside in an
> earlier reply still needs asking, unless the developer's instructions
> dropped it.
>
> If the developer could turn the doc into tickets without asking its
> authors anything more, there are no gaps left: send `doc_update` with
> `no_gaps` and the whole new body. Keep the authors' headings and their own
> sentences wherever they still hold, weave each answer into the section it
> belongs to, add a section only where nothing fits, and never mention
> questions or rounds. Add only what an answer says: never invent details,
> figures, examples, or requirements the authors did not give. Only
> questions that were asked and skipped twice, and still matter, go under a
> short "Open points" section.
>
> Never mention AI, an agent, or yourself in anything the authors see. Reply
> to the developer with what you asked and why, or that the doc is complete
> and what changed, plus any technical questions for them.
