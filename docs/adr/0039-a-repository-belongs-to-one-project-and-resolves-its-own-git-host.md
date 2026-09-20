# A repository belongs to exactly one project, and that link is how a git host is resolved

Linking a repository to a second project is rejected as a conflict. That is
deliberate rather than an oversight: `owner/name` is the only key a webhook
delivery, a pull-request event, and a repository scan arrive with, so a
repository that belonged to two projects would leave every inbound event
without a unique home.

The same link is what makes multiple git hosts work. A repository record
carries the id of the connector hosting it, and the composition root's router
resolves a provider per repository — look up the link, read that connector's
live token and base URL, build the client fresh for that call. Several git
hosts connected at once is the expected case, not an edge case, so there is no
single "active git provider" anywhere and a repository whose connector has
been disconnected fails on its own without affecting repositories on other
hosts.

Decided: 2026-09-03
