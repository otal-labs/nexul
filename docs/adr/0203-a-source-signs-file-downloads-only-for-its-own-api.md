# A source signs file downloads only for its own API, and people without an email match by username and name

Amends ADR 0201 for the Trello and Asana sources.

Trello serves an uploaded attachment only to a request signed with the API key and token, so ADR 0201's rule that a
file never carries the credential would leave every Trello upload a link. Decision: a source may sign a download, and
only one whose host is its own API's (`trello.com`, `api.trello.com` for Trello). The URL still has to be one of the
source's file hosts, every hop is checked, and Go drops the `Authorization` header on a redirect to another domain, so
the credential only ever goes back to the service that issued it. A link someone pasted into a card that points
elsewhere is fetched unsigned, as before. Asana's `download_url` is presigned and expires within minutes, so its files
come down unsigned, and each task's attachments are listed last, right before the task is handed to the sink, which
copies them at once.

Trello gives no emails, so a person who carries only a username matches a member whose login is that username, then
the one member whose name or display name is theirs; two members with the same name match nobody. A source with a
GitHub login or an email never falls back to names, since a wrong match would assign a ticket to the wrong person.

Rejected: Trello's key and token in the download URL (Trello takes them in the header, and a URL ends up in logs);
matching Trello people by name alone (a username is the stronger signal and is tried first).

Decided 2026-11-18.
