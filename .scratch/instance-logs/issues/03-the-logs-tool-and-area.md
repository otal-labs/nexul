# 03: The Logs instance area and how an agent reads the instance's logs

Type: grilling
Status: open
Blocked by: 01

## Question

Logs becomes an instance area: the instance Owner holds it by right, and its
Access card hands it to a role or a person (`instanceAreas` in
`internal/platform/permissions/permissions.go`). What exactly does it gate,
and what does an agent call?

- The action or actions (`logs:read` alone?) and their label in the catalog.
- The MCP surface, within the tool budget (`practices/mcp.md`, section 4):
  a `logs` query on `instance_get`, beside how `stack_get` takes `logs`, or a
  new tool that earns its place. Name it and its parameters: source, level,
  time range, `trace_id`, text, limit.
- The HTTP route the browser and integrations reach, even with no page for
  now (ADR 0019), and the gateway scope it needs.
- What comes back: the fields per line, the cap on lines and bytes, and how
  paging works over the search API ticket 01 describes.
- Redaction before a line leaves the server: tokens and secrets in messages
  and attributes. Browser logs are already redacted (`web/src/lib/browserLogs.ts`);
  server attributes are not. Who holds Logs matters here, because it need
  not be the Owner.
- The answer when OpenObserve is down or not installed: an error that says
  so, not an empty list.
- What the logs guide says afterwards. Does OpenObserve's own MCP stay
  documented as the Owner's road, or drop out of the guide?

Technical: answered in this ticket, not grilled with the owner.
