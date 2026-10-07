---
title: Repository Layout
description: What lives where in the Nexul monorepo.
sidebar:
  order: 2
---

Everything ships from one repository. This is what each top-level directory
holds.

| Path | What lives there |
|---|---|
| `cmd/nexul/` | The `nexul` command that installs, upgrades and removes Nexul's services on a machine; its logic lives in `internal/install/`. |
| `server/` | The `nexul-server` binary's composition root (`cmd/main.go`) and its development Dockerfiles; wires domains and platform packages together. |
| `runner/` | The `nexul-runner` binary — connects out to the server over WebSocket and executes builds and deploys on its host. |
| `internal/` | Every domain package (`internal/<domain>/`) plus `internal/platform/` (eventbus, storage, logging, config) shared by the binaries. |
| `web/` | The React app served to the browser — Vite, Tailwind, shadcn/ui — talking to the server over the HTTP/JSON gateway and a WebSocket. |
| `desktop/` | The Electron shell that wraps the served web app for a native-feeling install. |
| `sdk/` | The TypeScript package automations are written against — client, API client, generated event types, config schema. |
| `automations/` | The automations host, compiled into the `nexul-automations` binary, that runs the automations placed on it. |
| `docs/` | `adr/` — every durable decision, one file each; `agents/` — how the tracker and the glossary are used. The glossary itself is `CONTEXT.md` at the root. |
| `.scratch/` | The issue tracker — markdown files committed to the repo, one directory per effort, each with its spec and tickets. |
| `website/` | This documentation site — Astro with Fumadocs UI. |
| `practices/` | The coding standard, one file per language or surface. `AGENTS.md` routes each task to the file it needs. |

See [Architecture](/docs/contributing/architecture/) for how these pieces
relate to each other, and [Coding Standards](/docs/contributing/coding-standards/)
for the rules that govern what goes inside `server/`, `internal/`, and `web/`.
