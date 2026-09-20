# Attachment bytes live in the SQLite file, not on disk or in object storage

An attachment's bytes are stored in SQLite alongside everything else, capped
at 10 MB per file. A self-hosted instance then has exactly one backup story —
copy the database file — instead of a database plus a blob directory that can
drift out of sync with it, and no S3-shaped dependency for a product whose
whole premise is that it runs on your own server. The cap is what makes this
safe; large artifacts are not an attachment use case.

An attachment is owned by exactly one doc, ticket, or conversation, never
shared between them, so deleting the owner cascades cleanly and no
reference-counting is needed. Content type is sniffed server-side, not taken
from the upload: only raster images are served inline, and everything else —
including SVG, which can carry scripts — is served as a download with
`nosniff`. Because the gateway authenticates with a bearer token, the browser
cannot use a bare `<img src>` and fetches image bytes through the API client
to render an object URL; the file route must stay authenticated.

Attachments deliberately have **no MCP tools**, which is worth stating because
every other capability gets them by construction. Listing and uploading stay
browser-only until an agent actually needs them: markdown export already
renders each file as `![name](/api/attachments/<id>)`, so an agent reading a
doc or ticket can already follow its attachments, and an upload tool would
have to carry bytes over the tool protocol to earn its place.

Decided 2026-08-28.
