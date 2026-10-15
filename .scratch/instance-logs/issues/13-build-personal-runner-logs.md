# 13: Build personal runner logs and the share's Read logs

Type: task
Status: open
Blocked by: None — can start immediately

## What to build

Ticket 04, as decided: a personal runner writes its own rotated `runner.log` on every OS; a
`logs_request` with `target: runner` tails the file and follows the logger;
`computers:read_logs` on a computer's share covers the runner's and the containers' logs, off by
default and masked for sharees.

## Surfaces

- MCP: `computer_list` reads the runner's logs beside `container_logs`.
- Permission: `computers:read_logs` in the catalog and the share dialog.
- Docs: the computers guide's sharing section.
