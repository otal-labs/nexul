# The MCP adapter runs on the official Go SDK

Supersedes ADR 0048. The protocol and transport layer of `internal/mcp/` is
the stateless Streamable HTTP handler from
`github.com/modelcontextprotocol/go-sdk`, not code of our own. The tool
contract stays ours: domains declare tools through `internal/platform/mcptool`,
and only `internal/mcp/` imports the SDK, so an SDK upgrade touches one
package.

The hand-rolled server fell behind the 2026-07-28 revision within weeks of
adopting it: protocol errors went out with the wrong HTTP statuses, required
headers went unchecked, resource templates had no list method, and every tool
failure was a protocol error the model never saw. The specification ships a
revision every few months, each one changing wire details that are easy to
get subtly wrong, so keeping our own implementation conformant is recurring
work nobody schedules. The official SDK is held to
full conformance by the specification's SDK tiering and moves with each
revision, so conformance arrives as a dependency bump.

The cost is a direct dependency with its own transitive modules, and wire
behavior we no longer write: a protocol bug is fixed upstream or by
upgrading, not patched locally. The SDK negotiates the legacy handshake
revisions itself, so the dual-revision support ADR 0048 introduced stays,
without code of ours behind it.

Rejected at the same time, because the SDK made the question explicit:

- A stdio transport and its proxy command. Every client the product
  documents speaks Streamable HTTP with a static bearer header, the proxy
  needed the server's signing secret on the user's machine, and nothing
  built or shipped it.
- Server-Sent Events responses. The server sends nothing while a call is in
  flight, so every response is one JSON body.
- A dedicated MCP listener. `/mcp` on the main listener is the one endpoint,
  which also stops a second port from serving tokens over plain HTTP.
