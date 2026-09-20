# Runners update themselves: server-pushed, re-exec in place

A runner reports its version on every connect. When the server is a real release build and a connected runner
reports a different real version, the server sends it an `update` frame naming the version, a download URL back
to the server's own download route, and the release's sha256. The runner downloads the matching binary,
verifies the checksum, renames the running binary aside (`<exe>.old`), swaps the new one into place, and
re-execs itself in place (`syscall.Exec` on Unix; start-and-exit on Windows, which has no equivalent). A runner
mid-job defers the swap until its job's result frame has been sent. The bundled `instance` runner and any runner
running inside a container never apply this — they follow the compose image tag instead.

Rejected: refusing connections from an old runner and making the operator fix it by hand — that turns every
runner release into a support ticket, and the whole point of shipping a runner binary is that it runs
unattended on machines nobody wants to log into regularly. Also rejected: re-running the install command per
machine, driven from the server — that needs SSH access to every runner's host, which is exactly the credential
the runner model was built to avoid (the server holds no SSH keys and never connects into a target). Also
rejected: a separate updater process that supervises the runner and swaps its binary from outside — correct in
principle, but it doubles the thing to install and keep running per machine for a problem the runner can solve
for itself with a download and a rename.

The trade-off: a runner that dies mid-swap (killed before the final rename, or the new binary starts and
immediately crashes) is not self-healing — there is no supervisor watching it. The checksum step catches a
corrupted download before anything is touched, but a binary that downloads intact, passes verification, and
still crashes on start has no automatic rollback; `<exe>.old` sits right next to it as the manual recovery.
Accepted because the failure mode is loud (the runner simply stops reconnecting) and the fix is a one-line
rename back, not a redeploy.
