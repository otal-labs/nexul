# A host with no systemd runs the computer's runner and T3 Code under a built-in supervisor

Amends ADR 0073 and ADR 0146.

`nexul install computer` registered a systemd system unit, and T3 Code's own systemd user service, and refused a host
that was not running systemd. A container whose PID 1 is tini, with no user session bus and only outbound 443, could not
install either. The person enrolled by hand through `POST /api/runners/enroll` and kept the runner and `t3 serve` alive
with two shell loops that restarted each one after it exited, and every restart showed the computer as Offline.

Decision: on such a host the install registers nothing and runs a small supervisor in the foreground, `nexul run
computer`, which starts the runner and `t3 serve` as its children, starts either one again 5 seconds after it exits,
forwards SIGINT and SIGTERM to both and exits once they have stopped, and logs through `slog`. It is the unit's
`Restart=always` and `RestartSec=5`, kept by the `nexul` command because no service manager is there to keep it.

- **Detection.** The install runs the supervisor when `Paths.SystemdProbe` (`/run/systemd/system`, the `sd_booted` test)
  is absent on Linux, or when `--foreground` says so. macOS (launchd) and Windows (the logon task, ADR 0146) are
  unchanged, and `--foreground` there is refused.
- **The account is whoever runs the command.** Nothing it installs needs root, so there is no `sudo`, no chown and no
  root cleanup. Under `sudo` it refuses, because the runner would be someone else's; a root login in a container runs the
  runner as root, which ADR 0146 refused for a system service but which here is just the account that started it.
  `computer.sh --foreground` skips its sudo checks for the same reason.
- **Everything else is as installed today.** The command, the runner, its credential, `env` and `unit.json` land under
  the account's home. `unit.json` gains `foreground` and `t3`, the command line the supervisor runs T3 Code with
  (`t3 serve --host 127.0.0.1 --port <port>`, loopback only). The installer reuses a T3 Code that already answers and
  leaves it unsupervised, runs T3 Code's own installer when there is none, and skips both with `--no-t3`. It adds no
  library: `libatomic1` needs root, so the person installs it first.
- **The install ends by running the supervisor** and prints once the line that starts it again after a reboot, with a
  hint to put it in a container entrypoint or an `@reboot` cron entry. A lock file in the runner's folder holds the one
  supervisor per account; a second `nexul run computer` names the pid that holds it.

How the self-update works under it (ADR 0052). The runner swaps its binary and re-execs in place with `syscall.Exec`,
which keeps the pid, so the supervisor's `Wait` never returns and nothing starts a second runner; its stop signal reaches
the new image. A supervisor never restarts on a pid change, because there is none. A restart runs the runner binary on
disk, which the update already replaced. The supervisor's own binary is the `nexul` command in the account's
`~/.local/bin`, which the runner never touches: it changes only when a new one is put there, a running supervisor keeps
the image it started with, and the next `nexul run computer` runs the file on disk, so that start is how it moves on.
Windows's `NEXUL_SUPERVISED` exit-and-restart protocol is not used here, because in-place exec needs no help.

How removal works under it. A removed runner calls `nexul uninstall computer --detach`, which on a foreground install
writes `remove-requested` in the runner's folder and returns, where a service install asks root's path unit. The runner
then exits 0. The supervisor asks after each runner exit whether that file exists: if so it stops T3 Code, deletes the
folder and the `nexul` command and exits; if not, the exit is restarted like any other, so a stray SIGTERM to the runner
never wipes an install. An exit code is never the signal, which is why the file exists. `nexul uninstall computer` by
hand stops a running supervisor through its lock, then removes the same files and tells the instance.

How T3 Code restarts under it. The runner's keeper restarts a T3 Code that stopped answering with `t3 service restart`,
which needs a user session bus that this host lacks and whose failure reads "turn on lingering". The supervisor sets
`NEXUL_T3_SUPERVISED` on the runner, and the keeper then never restarts, so it reports no error for a restart that is
the supervisor's job.

What it does not do. It starts nothing at boot: the entrypoint or cron entry that runs it is the person's. A T3 Code that
hangs without exiting is not restarted, since the supervisor sees only exits. A supervisor killed with SIGKILL leaves its
children running, because the one kernel way to take them with it, `Pdeathsig`, fires when the thread that forked them
ends and a Go program cannot hold that thread; the lock stops a second supervisor but not the orphans, so the person
stops them first. On Linux `nexul status` and `nexul upgrade` do not list or move a computer, with or without systemd,
since both read the server's unit directory, so the `nexul` command there stays as installed until someone replaces it.

Rejected: requiring systemd, which leaves every container to the shell loops this replaces. Also rejected: installing a
third-party init (tini does not restart, s6 and runit are another install per host), which ADR 0073 already refused for
a supervisor of our own, a refusal that rests on OS service managers doing the job and so stops holding where there is
none. Also rejected: a restart policy in the runner itself, which cannot restart T3 Code and cannot restart itself when
it crashes.

Decided 2026-10-21.
