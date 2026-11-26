# A terminal is a pty on the runner, carried by its own stream

Amends ADR 0209 (its "no input").

A command from Nexul ran once with no input, so a person at the Commands section could not `cd` and stay there, answer
a prompt, or run `top` or `vim`. They wanted a terminal like the one on the computer itself.

Decision: the runner opens the default shell (bash as a login shell, else sh; pwsh, else Windows PowerShell) on a
pseudo-terminal (a Unix pty, ConPTY on Windows, through `github.com/aymanbagabas/go-pty`) in the home folder of the user
it runs as, without `NEXUL_*` variables and with `TERM=xterm-256color`. The server asks with `terminal_open {id, cols,
rows}`; the runner answers by opening the same per-connection stream a T3 Code dial uses (ADR 0146), or with
`harness_dial_refused`. On that stream binary messages are the terminal's bytes both ways, a text message towards the
runner is a size (`{"cols","rows"}`) and one from it is the shell's exit (`{"exit": n}`) before it closes. Either end
closing kills the shell's whole process tree.

- **Opening is a write, attaching is a socket.** `POST …/terminals` opens it on the runner and records it, so a refusal
  answers the request and the audit row is written like a command's; `GET /ws/…/terminals/{id}` then attaches the
  browser, once, for the person who opened it, within 30 seconds, after which an unattached terminal is closed.
- **The same gate as a command.** Both switches, the computer's owner or a share with Run commands, a machine's
  `machines:shell`; turning Nexul's switch off or losing the share closes the terminal. Four per runner, apart from the
  four commands.
- **Recorded as a job.** A row in `runner_shell_jobs` with adapter `terminal`: who opened it, when, and how it ended (the
  shell's exit code, closed by its viewer, or the runner gone), with the last MiB of what it printed under the same 30-day
  output purge. Keystrokes are never stored; a password typed at a prompt is not echoed, so it is not in the output.
- **The browser draws it with xterm.js,** loaded only when a terminal opens.
- **No MCP tool.** An agent cannot hold a screen; `command_run` stays its way to run something there.

Rejected: a terminal-looking dialog over one-shot commands (no prompts, no full-screen programs, `cd` faked by tracking a
folder), and multiplexing terminal bytes over the runner's control connection, where a busy terminal would delay its
heartbeats.

Decided: 2026-11-26.
