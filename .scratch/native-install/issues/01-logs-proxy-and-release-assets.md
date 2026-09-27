# 01 — `/openobserve/` proxy, component-named release assets, no images

**Status:** resolved
**Type:** task
**Blocked by:** None — can start immediately

## Scope

- Server: `NEXUL_LOGS_URL` in `internal/platform/config`; when set, mount a
  `httputil.ReverseProxy` at `/openobserve/` (path passed through unchanged,
  `FlushInterval: -1`, WebSocket upgrades pass through). It sits outside Nexul
  auth, and the web UI handler's reserved prefixes include `/openobserve` so
  the SPA fallback never answers it. Unset means no route.
- Release: `.goreleaser.yaml` builds `nexul` from `./cmd/nexul`,
  `nexul-server` from `./server/cmd` (`-tags embed`), `nexul-runner`; names
  `{{ .Binary }}-{{ .Os }}-{{ .Arch }}`. `release.yml` runs
  `bun run --cwd automations build:binaries` before goreleaser and hands the
  five `automations/dist/nexul-automations-*` files to goreleaser as release
  and checksum extra files. Remove `dockers_v2`, the `automations-image` and
  `prune` jobs, `server/Dockerfile.release`, `runner/Dockerfile.release`, and
  the image steps in `ci.yml`; keep `goreleaser check`.
- `./cmd/nexul` and the `build:binaries` script are created by tickets 03 and
  04; build against the contract, do not create them here.
- `website/public/install.ps1`: self-elevate when not admin, install `nexul.exe`
  to `%ProgramFiles%\Nexul` on the machine PATH, pass all arguments through to
  `nexul install`. `install.sh` already passes arguments through; confirm the
  `runner …` form works and extend `website/test/installers.test.ts` for it.
- Makefile: `build-server` outputs `dist/nexul-server`; add `build-cli` for
  `dist/nexul`.

## Acceptance

- Proxy tests: forwards path and query unchanged, streams, returns 502 when the
  upstream is down, absent when unset, never reaches the SPA fallback.
- `goreleaser check` passes; `bun run test` in `website/` passes.
