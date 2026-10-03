# The MCP tool ceiling rises to 107 for attachments

People attach images and files to docs, tickets, chat, and memories, but no tool could upload one or read one, so an
agent asked to put a client's logo on a ticket could only link the chat's files, and could not look at a screenshot
it was shown (ADR 0019). The server stood at 104 tools under the ceiling of 105 that ADR 0094 set.

Folding came first and found no home. An upload is a child of four different owners, and the agent needs its link
before it writes the body or message that embeds it, so it cannot be a field on `doc_update`, `ticket_update`,
`memory_update`, and `message_post` without four copies of one capability and a second call to embed it anyway.
Reading returns the file's bytes, which no `get` of its owner should carry, and ADR 0068 keeps reading and changing in
separate tools.

Decision: `attachment_create` uploads base64 content, or copies an existing attachment on the server, onto exactly one
owner through the same use-case as the browser's upload, so the permission check, the 10 MiB cap, and the content type
read from the bytes are the same. It returns the markdown the browser writes for an upload. `attachment_get` returns
the metadata and the contents, as an image block for the formats models read, text for text, and a base64 resource
otherwise. The MCP request body cap rises from the SDK's 4 MiB to fit a whole upload as base64. The ceiling becomes
107, exactly enough for the 106 tools the server now has.

The trade-off: two more definitions in every session, and a client that caps an agent at 100 tools across all its
servers drops two more of Nexul's. Accepted over leaving files to the browser alone.

Rejected: reading a path on the paired computer, which the server cannot reach; agents send the bytes, or copy a file
people already posted.

Decided 2026-10-03, amending ADR 0094's tool ceiling.
