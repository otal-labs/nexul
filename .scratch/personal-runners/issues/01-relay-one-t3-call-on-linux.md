# 01 — Reach T3 Code through a personal runner's connection, proven with one call on Linux

**Status:** resolved

**Blocked by:** None — can start immediately

Read first: `practices/go.md` (sections 6, 7, 13, 14, 15), `practices/architecture.md` (sections 5, 11),
`practices/testing.md`, ADRs 0031, 0052, 0074, 0091, the spec (The relay, The runner protocol, Security model).

## What to build

The smallest end-to-end proof that the server can reach a computer's T3 Code through the runner, with no
UI and no pairing yet.

- Migration: `runners.owner_user_id` and `runners.computer_id`, both `TEXT NOT NULL DEFAULT ''`, and the
  partial index on `computer_id` (comment naming the lookup it serves). No backfill.
- Runner, personal mode (`NEXUL_RUNNER_MODE=personal`): finds T3 Code's port (`server-runtime.json`, else
  3773); answers `harness_dial {id}` by dialing loopback on that port and opening
  `/api/runners/streams/{id}` with its credential, or `harness_dial_refused {id, error}`; copies bytes both
  ways through `websocket.NetConn(…, MessageBinary)`; caps 32 open streams; refuses `assign_build`,
  `assign_deploy`, `assign_upgrade`, `join_networks`, `discover`, `logs_request` with a warning.
- Server: `runner.Handler.DialComputer(ctx, computerID) (net.Conn, error)`, the stream endpoint (single-use
  id, 10 s life, accepted only from the runner it was issued to), 32-stream cap, streams closed on revoke.
  `fits`, `connOnMachine` and `connNamed` skip personal runners.
- The harness `http.Client` (`server/cmd/services.go:221`) gets a `DialContext` that sends
  `<computer id>.nexul-computer.invalid` hosts to `DialComputer` and sets `Host` to the loopback address;
  everything else dials as today, still wrapped by `AccessTransport`.
- The frames get validators in `protocol.go` and table rows in `protocol_test.go`.

For this slice a test marks a runner personal by writing the two columns; enrollment is ticket 02.

## Acceptance criteria

- [x] An integration test in `server/cmd` connects an in-process personal runner to a real server against
      a `t3rpctest` fake T3 on loopback, and `t3rpc.Describe` through the harness client returns the fake's
      descriptor.
- [x] Through the same relay, a `t3rpc.Connect` session completes `server.getConfig`, and a second HTTP
      call reuses the pooled connection (one stream opened, counted).
- [x] A relayed WebSocket held idle for 30 minutes under `testing/synctest` (or a real-time soak run once and
      reported in the PR) stays open; this settles the `NetConn` deadline risk in the spec.
- [x] `TestDialComputer_RefusesAStreamFromAnotherRunner`, `TestDialComputer_StreamIDIsSingleUse`,
      `TestDialComputer_ExpiredStreamIsRefused`.
- [x] `TestPersonalRunner_RefusesDeployFrames` and `TestDispatch_NeverPicksAPersonalRunner` (a job with no
      target machine stays queued while only a personal runner is connected).
- [x] The runner never dials anything but loopback: a test with T3 Code's runtime file naming a LAN address
      gets `harness_dial_refused` and no connection attempt.
- [x] Manual check on Linux, reported in the PR: a `nexul-runner` started by hand in personal mode against
      a local server, with a real T3 Code bound to `127.0.0.1`, answers a relayed descriptor request.
- [x] `TestRunner_APanicInOneStreamLeavesTheOthersRunning`: a relayed stream whose copy panics closes alone;
      the control connection and another stream keep working (the spec's Lanes).
- [x] No new `require` in `go.mod`; `goleak` stays clean in `internal/runner`.

## Comments

Built: migration 0091 (`runners.owner_user_id`, `runners.computer_id`, partial index serving `GetRunnerByComputer`),
`internal/runner/streams.go` (both ends), `harnessHTTPClient` in `server/cmd/wire_computer_streams.go`, and
`GET /api/runners/streams/{id}` beside `/ws/runner`.

- A personal runner answers the six deploy-side frames with their failed terminal frame (`deploy_result`,
  `upgrade_result`, `join_networks_result`, `discover_result`, `logs_end`) as well as a warning, so a server that
  ever sent one fails fast instead of waiting.
- The 30-minute idle check is `TestStream_IdleWebSocketStaysOpenForThirtyMinutes` under `testing/synctest`, over
  in-memory pipes through the real stream endpoint, `websocket.NetConn` on both ends and `http.Transport`.
- Go pools an upgrade request apart from plain requests, so a `t3rpc.Connect` takes its own stream; plain calls
  share one (counted in `TestIntegration_PersonalRunnerReachesT3Code`).
- T3 Code deletes `server-runtime.json` when it stops. Only the default home (`~/.t3`) falls back to 3773; a
  custom `T3CODE_HOME` with no file refuses the dial (`TestPersonalRunner_CustomHomeWithoutARuntimeFileIsRefused`),
  so a recipe pointed at a throwaway never reaches the developer's own T3 Code.
- For 02: `POST /api/runners/enroll` still makes a machine row for every runner; a personal runner skips it.
  Personal runners still show in `GET /api/runners` and still publish `runner.connected`/`disconnected`.
- The relayed request's `Host` is `127.0.0.1` without a port, since only the runner knows the port.
