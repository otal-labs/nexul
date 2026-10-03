# 04 — Record protocol-2 frames that need no provider

**What to build:** Real protocol-2 frames as test fixtures, so ticket 05 parses what T3 sends, not
only what its source says.

1. `incus copy nexul-box/clean nexul-box-t3 && incus start nexul-box-t3`. The box has no internet:
   download the release archives on the host (`t3-0.0.45-linux-x64.tar.gz` and
   `t3-0.0.46-nightly.20261003.2632-linux-x64.tar.gz` with their `SHA256SUMS`), serve them with
   `python3 -m http.server` laid out as `v<ver>/…`, and install with `T3CODE_RELEASE_BASE_URL` and
   `T3CODE_VERSION` set. Nightly `2632` is the first build containing upstream 6108ef3d3d; a later
   one is fine if `packages/contracts/src/orchestrationV2.ts` is unchanged against `31a9da17` for
   the fields Nexul reads.
2. Under a `t3` user, each version with its own `T3CODE_HOME`: `t3 serve --host 0.0.0.0 --port 3773`
   (0.0.45) and `--port 3774` (nightly), `T3CODE_TELEMETRY_ENABLED=false`. Get a bearer per server:
   `t3 pair --base-dir <home>`, then exchange the token at `/oauth/token` as Nexul does. Register a
   project: `t3 project add <dir> --base-dir <home>` (or `POST /api/projects/mutate` on the nightly).
3. Capture with a throwaway client kept outside the repo: the 426 body; the descriptor and
   `server.getConfig` on both; the shell snapshot; `thread.create` and its first snapshot; a
   `message.dispatch` that fails on an unauthenticated provider (the run failure shape and the root
   error item); `getThreadProjection`; `queued-run.cancel`; a subscribe resume with `afterSequence`;
   and a subscribe to a deleted thread.
4. Commit as `internal/t3clientv2/testdata/<scenario>.0.0.46-nightly.20261003.2632.ndjson` (and
   `.0.0.45` for protocol-1 ones), ids rewritten to `th-1` style, no tokens, paths, hostnames or
   emails. Write what the captures settle, and anything that differs from `research/protocol-2-wire.md`,
   into a "Findings" section here.
5. `incus stop nexul-box-t3 && incus publish nexul-box-t3 --alias t3-dual`, then `incus delete
   nexul-box-t3`. Ticket 16 launches from the image.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

Read first: the spec, `research/protocol-2-wire.md`, and the Incus box notes in the owner's memory.
Never touch the host's port 3773 or the owner's `~/.t3`.

- [ ] One fixture per scenario above, named after its T3 build
- [ ] `rg --hidden '@|/srv/|/home/|Bearer|wsTicket' internal/t3clientv2/testdata` finds nothing
- [ ] Findings written; `incus image list` shows `t3-dual`; the clone is deleted
