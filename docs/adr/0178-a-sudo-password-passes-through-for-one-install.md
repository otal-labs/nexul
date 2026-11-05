# A sudo password passes through Nexul for one install, and never through an agent

Installing git or gh on Linux ran `sudo -n`, so on a computer whose sudo asks for a password, which is most of them,
the Install button could only say what to run in a terminal there. A computer that has just joined often lacks git,
and setup's skill install needs it.

Decision: when `sudo -n` says it needs a password, the web asks for the person's sudo password and sends it with the
install. The server passes it to the computer's Nexul app in the `install_tool` frame, which runs
`sudo -k -S -p ""` with the password on stdin; a wrong one asks again.

- **Held for one request only.** The password is in the request body and the frame, never in a row, an event, an
  audit entry (they record the method and path) or a log line. The error that comes back is sudo's verdict, never
  the input.
- **`-k` makes sudo read it every time**, so the password never reaches the package manager's stdin when sudo
  remembers an earlier one.
- **Not through MCP.** `computer_pair` with `install_tool` takes no password: an agent would see it in its own
  context. Its refusal points at the button, or the command to run on the computer.
- **An older app is refused** before the password is sent, since it would drop the field and run `sudo -n` again.

Rejected: a sudoers rule for the runner, written at install time. It gives the runner standing root for package
installs, which is more than one install needs, and would be done by the install script under the person's sudo
without them choosing it.
