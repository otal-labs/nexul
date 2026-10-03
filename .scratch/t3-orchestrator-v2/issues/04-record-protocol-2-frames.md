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

**Status:** done

Read first: the spec, `research/protocol-2-wire.md`, and the Incus box notes in the owner's memory.
Never touch the host's port 3773 or the owner's `~/.t3`.

- [x] One fixture per scenario above, named after its T3 build
- [x] `rg --hidden '@|/srv/|/home/|Bearer|wsTicket' internal/t3clientv2/testdata` finds nothing
- [x] Findings written; `incus image list` shows `t3-dual`; the clone is deleted

## Findings

Captured 2026-10-03 from `0.0.45` and `0.0.46-nightly.20261003.2632`. The nightly is one commit past
`31a9da17`, with no change under `packages/contracts/src` or in `apps/server/src/ws.ts`.

**The fixtures.** One JSON record per line, in the order it happened on one connection:

- `{"dir":"http","method","path","status","body"}`: a plain HTTP GET.
- `{"dir":"dial","path":"/ws","query"}`: the WebSocket dial, with the ticket left out.
- `{"dir":"send","frame"}` and `{"dir":"recv","frame"}`: Effect RPC frames as sent and received.
  Ping, Pong and Ack are not recorded.

Ids are rewritten per file in order of appearance:

| Prefix | Id |
|---|---|
| `th-` | thread |
| `pr-` | project |
| `msg-` | message |
| `cmd-` | command |
| `ev-` | event |
| `ps-` | provider session |
| `native-` | provider-native id |
| `env-` | environment |

T3's composite ids keep their shape around the rewritten parts, for example
`run:thread:th-1:ordinal:1` and `turn-item:message:msg-1`. Paths become `/workspace`, `/t3-home` and
`/user-home`, and the hostname becomes `t3-host`.

| Fixture | What it holds |
|---|---|
| `protocol-mismatch` | `GET /ws` with no protocol, `=1` and `=3` |
| `descriptor` (both builds) | the well-known descriptor, then a dial with `orchestrationProtocol=2` and `server.getConfig` |
| `subscribe-shell` (both builds) | the shell snapshot with one project |
| `thread-create` | `thread.create` and the first bounded snapshot |
| `dispatch-provider-unauthenticated` | Claude 2.1.288 installed but not logged in: dispatch, then the stream until the run fails |
| `dispatch-provider-missing` | the same with no `claude` binary |
| `get-thread-projection` | the unauthenticated thread, then a thread id that does not exist |
| `queued-run-cancel` | a `defer_start` blocker in `preparing`, a run queued behind it, `queued-run.cancel`, a second cancel, then `run.interrupt` on the blocker |
| `subscribe-resume` | resubscribe with `afterSequence` after a run happened unwatched |
| `subscribe-deleted-thread` | a live watch across `thread.delete`, a fresh subscribe, then `message.dispatch` on the deleted thread |
| `subscribe-missing-thread` | subscribe to a thread id that never existed |

**What the captures confirm in `research/protocol-2-wire.md`:**

- **The 426 body** is byte for byte as documented, for a missing, `1` or `3` protocol value.
  - The check runs before auth and before the upgrade, so a plain `GET /ws` gets it.
  - `0.0.45` answers the same GET with 401 `auth_invalid` / `missing_credential`.
- **Envelope and command results.** The Effect RPC envelope is unchanged, a command succeeds with
  `{"sequence":N}`, and a command failure is `OrchestrationV2DispatchCommandError` with `message`,
  `detail` and a nested `cause`.
- **A missing thread.** `subscribeThread` and `getThreadProjection` both fail with
  `OrchestrationV2GetThreadProjectionError`, which carries `threadId` and a cause chain ending in
  `ProjectionStoreThreadNotFoundError`.
- **Cancelling a queued run.** `queued-run.cancel` gives `run.updated` `cancelled`, with
  `queuePosition: null` and `completedAt` set. Cancelling again gives "Run <id> is not queued."
- **Resuming.** `afterSequence` replays event items only, with no snapshot; here that was 24 events
  in one Chunk. Sequences are global, so the gaps belong to other threads.
- **A failed run.** In both failure captures the root error item (status `failed`,
  `nodeId == run.rootNodeId`) arrives before `run.updated` `failed`. This settles the first "Still
  unknown" for Claude start and auth failures; other adapters are still open.

**What differs from the research or adds to it:**

1. **A deleted thread still takes messages.** `message.dispatch` on a soft-deleted thread succeeds,
   and T3 creates and starts a run on it.
   - So "the dispatch fails" never signals a deleted thread. Only the snapshot's `deletedAt` or a
     live `thread.deleted` event does. Ticket 05 must check `deletedAt` before dispatching.
   - The live stream does not end on delete: `thread.deleted` arrives (its payload is the AppThread,
     with `deletedAt` set) and the stream stays open.
2. **The dispatch reply can come first.** The `{"sequence":N}` Exit for a `message.dispatch` can
   arrive before the stream delivers that message's `run.created`.
3. **Optional keys are absent, not null.**
   - A new native thread has no `historyOrigin` key and no `limitRecovery`; treat absent as native.
   - A run has no `queueHeld` key unless it is true.
   - `delegatedCompletion`, `restartContinuationOfRunId` and `workStartedAt` are absent on ordinary
     runs.
4. **An unauthenticated Claude.**
   - The failed run first gets an `assistant_message`, status `completed`, with the CLI's text "Not
     logged in · Please run /login".
   - Then comes an `error` item: `class: "provider_error"`, `code: "api_error"`, message "Claude
     could not authenticate. For subscription login, run `claude auth login` on this environment's
     machine, …".
   - A failed run can therefore carry assistant text that must not become the reply. The root
     error's message is the one to show.
5. **A missing `claude` binary.**
   - `getConfig` reports the provider `installed: false`, `status: "error"`, but T3 still accepts the
     dispatch.
   - The session goes `ready`, then the turn fails as `transport_error` with "Provider turn failed."
     and `code: null`.
   - In both failure captures the provider session's `lastError` stays `null`, so the `lastError`
     fallback adds nothing for these failures.
6. **`auth.status` cannot be trusted.** `server.getConfig` reports Claude `auth.status:
   "authenticated"` on a machine with no Claude login.
7. **Closing your own stream.** An Interrupt is answered with an Exit `Failure` whose cause is
   `[{"_tag":"Interrupt","fiberId":N}]`. The client must treat that as a normal close, not a stream
   failure.
8. **Interrupting a run that has not started.** `run.interrupt` on a `preparing` run emits
   `run_interrupt_request` (`message` is the reason sent), then `run_interrupt_result`, status
   `interrupted`, with "Run interrupted before provider start". After those comes `run.updated`
   `interrupted`.
9. **`defer_start` really waits.** It leaves the run in `preparing`, with a `command_execution` item
   "Preparing workspace" in status `running`. That confirms Nexul never uses it.
10. **The shell snapshot changed slightly.** The nightly's shell snapshot has `schemaVersion: 2` (the
    research says 1), and a second snapshot follows with `resolvedRepositoryIdentityRoots`. Project
    rows have the same keys on both builds, and neither has `deletedAt`.
11. **`0.0.45` ignores the protocol query.** It accepts a dial carrying `orchestrationProtocol=2`
    and speaks protocol 1 on it. Its descriptor and `getConfig` `environment` both send
    `orchestrationProtocolVersion: 1` explicitly. The protocol-2 client must therefore refuse on the
    handshake's protocol (ticket 02), because the dial alone never fails.
12. **Smaller shapes.**
    - Provider events carry a `driver` field.
    - A bounded snapshot item also has `historyCursor`, `latestLocalTurnOrdinal` and
      `payloadBudgetExceeded`.
    - A user message's turn item id is `turn-item:message:<messageId>`.
    - `message.updated` with role `assistant` duplicates the `assistant_message` item, as the research
      says.

**Not captured.** Everything that needs a working provider is left for ticket 16:
- streamed assistant text, tools, questions and subagents;
- `waiting` and `completed`;
- a `LiveStreamBufferError` Die;
- a usage limit.

## Comments

- The `t3-dual` image runs T3 `0.0.45` on 3773 and the nightly on 3774. Each is an enabled systemd
  unit (`t3-45`, `t3-46`) under user `t3`, with `T3CODE_HOME` at `/home/t3/home45` and
  `/home/t3/home46`. Each has a `demo` project registered at `/home/t3/work4x/demo`.
- To pair, run `t3 pair --base-dir /home/t3/home4x` as `t3`; the binaries are in `/home/t3/bin45`
  and `/home/t3/bin46`.
- What else is in the image:
  - `libatomic1`, which the T3 release binaries need and the clean Ubuntu image lacks;
  - Claude Code 2.1.288 at `/usr/local/bin/claude`, not logged in, ready for ticket 16's provider
    login;
  - a few "Fixture thread" threads and one paired session from the capture client, in the nightly's
    state.
- The box has no internet, so archives and packages were served from the host.
- The capture client lives outside the repo in `t3code-research/capture`. To re-record, run
  `bun capture.ts <scenario>` against a box from the image, then `bun scrub.ts <testdata dir>`.
