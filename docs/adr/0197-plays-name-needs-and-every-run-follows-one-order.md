# Plays name needs, and every run follows one order

Amends ADR 0165 and ADR 0156.

Three lists decided where a run went: the person's order, a project's own list, and an auto play's chosen computers,
which replaced both for its runs. The one that won was not on the page a person looked at. On 2026-11-18 a Fix with AI
run waited on a computer named only by its auto play, and the fix link opened a project page that did not list it.

Separately, a project with an app and a tests repository found two checkouts on every computer that had cloned both,
and treated that as ambiguous everywhere.

Decision: one order, and plays say what they need instead of where to go.

- **One order.** A person's computers, in their order, are the only order. A project may narrow it as ADR 0165 allows.
  No play and no auto play carries a computer list. A pressed run may still pick one computer for that one press.
- **Needs.** A play names labels it needs. A computer has the labels its owner sets plus `docker` and its OS, derived
  from its facts. A run goes to the first computer in the order that has every need; one missing a need is passed over
  as `missing_need`. Needs belong to the play, so pressed and auto runs land under the same rule.
- **Run as a person.** ADR 0156's Run on a chosen computer becomes Run as a person: the run takes that person's identity
  and their order. The right checks ADR 0156 makes at pick, match and the front of the queue apply to the person.
- **No shared computers for auto plays.** With the list gone an auto play cannot name a computer shared with its person,
  and a shared computer is not in anyone's order. An auto play that named only shared computers is switched off on
  upgrade rather than moved onto its person's own machines.
- **The app repository wins a checkout tie.** Several matching checkouts on one computer resolve to the clone of the
  project's `app` repository; only two folders of the same repository are ambiguous.

The trade-off: a play can no longer say "only on that machine" by name. A machine that is the only one with something
gets a label for it, which is one more thing for an owner to set, but it survives a reinstall or a second machine with
the same thing, and the reason is written down where a name never said why. Losing shared computers for auto plays is
deliberate until a person can add one to their own order.

Rejected: keeping a named computer beside needs, which keeps two ways to say the same thing and the confusion with it;
per-play lists shown on the project page, which makes the override visible but keeps three lists; and dropping order
for "any ready computer that takes this project", which loses the main-machine-first rule ADR 0165 chose.

Decided 2026-11-18.
