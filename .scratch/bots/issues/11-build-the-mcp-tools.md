# 11 — MCP tools for bots

**What to build:** `botwebhook_list` (read-only; URLs only for a caller
with `botwebhook:write`), `botwebhook_create`, and `botwebhook_update`
(name, avatar, `regenerate`, `deleted` true or false) in
`internal/botwebhook/mcp.go`, over 09's use-cases, per the spec's
"Gateway, MCP, events" section. An ADR raises the ceiling in
`internal/mcp/surface_test.go` from 108 to 111, saying why no existing tool
can carry this (chat would have to learn what a webhook is). No tool posts
as a bot.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] Surface test at the new ceiling; tool descriptions follow
      `practices/mcp.md`
- [ ] A read-only caller's list carries no URL or token
- [ ] The ADR and `practices/mcp.md` updated together
