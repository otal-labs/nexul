# MCP clients sign in, and a grant is a personal access token row

Connecting an MCP client meant creating a personal access token and pasting it into the client. Clients and MCP
directories now expect the MCP authorization flow instead: the server's `401` names its protected-resource metadata, the
client registers itself, sends the person's browser to an authorize page, and trades a code for tokens.

Decision: the instance is its own OAuth authorization server for its MCP endpoint, and nothing more.

- **Public clients only.** Dynamic client registration (RFC 7591) is open, throttled per address, and issues no secret;
  every code needs PKCE with S256. A redirect is https, http on loopback (RFC 8252), or an app's own scheme, never one a
  browser would run, since the consent page follows it.
- **The person's own session decides.** The authorize endpoint is a web page: it signs the person in with whatever
  provider they use, shows the name the client registered beside where approving sends them, and only a browser
  session may approve, never a token or someone viewing as the person.
- **A grant is a personal access token row.** It carries the client, an access token that expires after an hour, and a
  refresh token that rotates on each use and is refused after 30 days unused. A spent code (for its first hour, while
  it is held in memory) or the refresh token the grant last rotated away from, presented again, revokes the grant
  (RFC 9700), since one of the two parties holding it is not the client; an older refresh token is only refused. So it lists, shows its last use and revokes exactly as a pasted token does,
  authenticates through the same path with the person's own permissions, and a revoke stops both its tokens at once.
- **A grant works only where it was issued, and mints nothing that outlives it.** Its token authenticates at `/mcp`
  alone, never the HTTP gateway or a live socket, and a tool that hands out a credential (a personal, computer or
  automation token, a computer's install command, an invitation link or a bot's webhook URL) refuses it. Otherwise a client could mint itself a token that
  survives the revoke.
- **Codes live in memory** for their minute. A restart mid-sign-in means approving again.

The trade-off: a grant holds every permission its person does, as a pasted token does; there are no scopes to narrow
it, because nothing in the permission table maps onto them yet. So what it changes as the person, such as a role or
another account's access, stays changed after the revoke; only the credentials it could hand itself are held back.
The tools that hand one out are marked one by one, so a new such tool is open to grants until it is marked. Behind a proxy every client shares one address, so the
registration throttle is shared too.

Rejected: a separate grants table, which would need its own list, revoke and authentication path beside the token one;
an authorize endpoint served by the server rather than the web app, which cannot see the browser's session; and
accepting `plain` PKCE or confidential clients, which MCP clients do not need.

Decided 2026-11-19.
