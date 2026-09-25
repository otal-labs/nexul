---
title: Logs
description: Where server logs go, and how to query them yourself or through the built-in MCP server.
sidebar:
  order: 8
---

An install made with `nexul install` includes [OpenObserve](https://openobserve.ai), and the server ships every log line to it as well as to stderr.

## Opening the logs UI

Open `http://<host>:5080` and sign in as `nexul@nexul.local`, with the password you chose when running the install script — or the one it generated for you, printed at the end of the install and saved to `.env` as `NEXUL_LOGS_PASSWORD`. Both the email and password can be changed later inside OpenObserve itself.

Logs land in the `nexul` stream of the `default` org.

## Environment variables

| Variable | What it sets |
| --- | --- |
| `NEXUL_LOGS_EMAIL` | The OpenObserve root user's email |
| `NEXUL_LOGS_PASSWORD` | The OpenObserve root user's password |
| `NEXUL_LOGS_TOKEN` | The ingest token the server uses to ship logs |

All three are generated into the install directory's `.env` by `nexul install` and reused on every re-run, because OpenObserve reads them only at its first boot.

## Querying logs through MCP

OpenObserve ships its own MCP server, so an agent can search your logs directly. It authenticates with the UI password, not the ingest token. To add it in Claude Code:

```sh
claude mcp add --transport http nexul-logs http://<host>:5080/api/default/mcp \
  --header "Authorization: Basic $(echo -n "$NEXUL_LOGS_EMAIL:$NEXUL_LOGS_PASSWORD" | base64 -w0)"
```

## Single-binary installs

A single-binary install has no bundled OpenObserve. Point it at any OTLP/HTTP backend instead:

| Variable | What it sets |
| --- | --- |
| `NEXUL_OTLP_ENDPOINT` | The backend's base URL (`/v1/logs` is appended automatically) |
| `NEXUL_OTLP_USER` | Basic auth username for the backend |
| `NEXUL_OTLP_TOKEN` | Basic auth token for the backend |

Leave `NEXUL_OTLP_ENDPOINT` unset to keep logs on stderr only.
