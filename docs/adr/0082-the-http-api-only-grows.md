# The HTTP API only grows

The browser is served by the same binary that answers it, so a route could be
renamed or a field dropped and the web app rebuilt in the same change. The
Android app breaks that: a phone runs whatever build it installed months ago
against whatever the instance was upgraded to since, and the owner's rule is
that the server is always the newer side. The app carries one minimum server
version and checks it on every launch through the public `GET /api/about`,
which returns only `{"product": "nexul", "version": "<tag>"}`, so a typed
address can be confirmed as a Nexul server before a connect code is spent.

Decision: a route or JSON field that a released app may call is never removed
or renamed. A replacement ships beside the old one, and the old one keeps
answering. The rule applies to the whole HTTP gateway, not only the routes
the phone uses today, because which routes a released build calls is not
knowable from the server side. It is the HTTP counterpart of ADR 0044, which
already holds the event catalog to additive-only.

The cost is that a naming mistake is permanent and the OpenAPI document
accumulates superseded routes. Accepted over a capability list or an API
version prefix, both of which would let the server refuse an old app that
would have worked, and over forcing a phone update on every server release.

Decided 2026-09-28.
