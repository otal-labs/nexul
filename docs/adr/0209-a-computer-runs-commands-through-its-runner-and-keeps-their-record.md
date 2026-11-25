# A computer runs commands through its runner and keeps their record

An agent on one computer could not test on another: a Windows-only bug was timed on Linux and the person who had it
was asked for numbers. A personal runner already holds a connection to its computer (ADR 0146), so it can run a
command there as the person it runs as.

Decision: a shell job is a typed runner job. The server sends `shell_run` (shell, command line, folder, timeout), the
runner runs it in bash, else sh, or in pwsh, else Windows PowerShell, with no input, no console window and no
`NEXUL_*` variables, streams `shell_output` in the `logs_chunk` shape at up to 64 KB a second, and ends with
`shell_result`. A timeout or a cancel kills the whole process tree. A disconnect fails the job.

- **Two switches, both needed.** The computer's own opt-in is its runner's unit setting: on when the person runs the
  install command, off with `--no-shell` or `nexul shell off`, and the server can never turn it back on. The owner's
  switch is on the computer row in Nexul; turning it off cancels the jobs there.
- **Owner only.** Starting, reading and cancelling go through the computer's ownership (and later its share), never a
  role; anyone else gets not found.
- **Four at a time per computer.** The rest wait in the server's memory and are checked again as they reach the front;
  a restart fails what was waiting or running.
- **Every job is recorded.** `runner_shell_jobs` keeps the command, who ran it, through which adapter, when and how it
  ended forever, beside but outside the audit log and its 45-day purge. The output, its last MiB, is emptied 30 days
  after the job ends by a daily loop.
- **Agents wait for it.** `command_run` is its own tool, an execution annotated destructive and open-world, capped at
  five minutes, returning the status, the exit code and the last 64 KB of output. `computer_list` with an id and
  `commands` reads the record back. The advertised size budget rises to 198,000 characters for it.

Read computer activity is how anyone besides the owner reads that record. It is an instance area, `computer_activity:read`, which the instance Owner holds by right like every instance area since ADR 0151, so no role gains it on upgrade and no migration grants it. The use-case that acts on a computer leaves a note on the request (`internal/platform/auditnote`), and the audit row the HTTP or MCP adapter writes after it carries the computer, its owner and the command's job. `GET /api/audit` leaves those rows out, and `GET /api/audit/computer-activity` joins them to the job for the command, its exit code and its output.

Rejected: running commands through T3 Code's own terminal, which needs a paired harness and a person's provider
sign-in, and leaves no record Nexul owns.

Decided: 2026-11-25.
