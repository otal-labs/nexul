# 02: Every one-way door

Type: research
Status: open
Blocked by: None — can start immediately

## Question

What can be created or changed but never removed, undone or edited again, from the web, the phone, HTTP or MCP?
Start from the four the walks found: a machine (no delete route), a Cloudflare tunnel (delete exists on the
server and MCP, not on the web), a stack's environment and secrets (editable only in the creation wizard), and a
failed deploy (no retry). Then compare every create against its delete or undo across every domain, and list:
- copy that promises a way back that doesn't exist ("Add it again to bring it back", "their computers are
  disconnected")
- states that can be entered but not left

For each finding, note what the server already supports, so the fix may be web-only.
