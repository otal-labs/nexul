# Code review is a mirror of the provider's review state, and provider events are its only writer

Nexul unifies the software lifecycle, so hosting diff review alongside
docs, tickets, and deploys is the expected move — it is an explicit non-goal.
Review happens on the git host; Nexul keeps one record per pull request
carrying a single aggregate status (`pending`, `approved`,
`changes_requested`, `merged`, `closed`, latest review wins) so agents can
find blocked work without calling the provider, and the ticket's development
panel deep-links out for the actual diff.

A review record hangs off a pull request and nothing else — it carries no
document link, because the chain already runs docs → tickets → pull requests
and a second edge from a review straight to a document would give the same
relationship two representations to keep in step.

Nothing but a normalized provider event may change that status: there is no
manual override route and no MCP tool for it, so an agent cannot approve its
own pull request. Gates stay soft — an unapproved review blocks no ticket and
no deploy; if enforcement is ever wanted it belongs to the domain doing the
gating, not to this one.
