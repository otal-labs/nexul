---
title: Logs
description: Where server logs go, and how to query them yourself or through the built-in MCP server.
sidebar:
  order: 8
---

An install made with `nexul install` includes [OpenObserve](https://openobserve.ai), and the server ships every log line to it as well as to stderr. OpenObserve runs as the `nexul-openobserve` service and listens on localhost only; the server reaches it there and serves its UI on your web port at `/openobserve/`, so there is no second port to open.

## Opening the logs UI

Open `http://<host>/openobserve/` (with `:<port>` when the web port isn't 80, and your `https://` instance URL once a proxy is in front) and sign in as `nexul@nexul.local` with the password the install generated. It is printed at the end of the install and saved to the install directory's `.env` as `NEXUL_LOGS_PASSWORD`. Both the email and password can be changed later inside OpenObserve itself. Nexul's own sign-in does not guard this path; OpenObserve's login does.

Logs land in the `nexul` stream of the `default` org.

## Environment variables

| Variable | What it sets |
| --- | --- |
| `NEXUL_LOGS_EMAIL` | The OpenObserve root user's email |
| `NEXUL_LOGS_PASSWORD` | The OpenObserve root user's password |
| `NEXUL_LOGS_TOKEN` | The ingest token the server uses to ship logs |

All three are generated into the install directory's `.env` by `nexul install` and reused on every re-run, because OpenObserve reads them only at its first boot.

## Querying logs through MCP

OpenObserve ships its own MCP server, so an agent can search your logs directly. Point your MCP client at the Streamable HTTP endpoint `http://<host>/openobserve/api/default/mcp` with a basic `Authorization` header built from the UI email and password, not the ingest token:

```sh
echo "Authorization: Basic $(printf '%s' "$NEXUL_LOGS_EMAIL:$NEXUL_LOGS_PASSWORD" | base64 | tr -d '\n')"
```

## A server run by hand

A server started on its own, without `nexul install`, has no bundled OpenObserve. Point it at any OTLP/HTTP backend instead:

| Variable | What it sets |
| --- | --- |
| `NEXUL_OTLP_ENDPOINT` | The backend's base URL (`/v1/logs` is appended automatically) |
| `NEXUL_OTLP_USER` | Basic auth username for the backend |
| `NEXUL_OTLP_TOKEN` | Basic auth token for the backend |

Leave `NEXUL_OTLP_ENDPOINT` unset to keep logs on stderr only.
