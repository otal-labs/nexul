# 13: Build personal runner logs and the share's Read logs

Type: task
Status: resolved
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

## Built

- A sharee without Read logs, or a stranger, gets 404 before Docker is checked; a sharee with it
  on a computer without Docker still gets 409 for containers, and the runner log opens anyway.
- ERROR records go out on the `stderr` stream, the rest on `stdout`.
- `computer_list` takes `runner_logs.lines`; masking fails closed when the env can't be read.
- No web viewer for the runner log yet: MCP, HTTP and the socket only.
