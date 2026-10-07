---
title: Logs
description: Read the instance's own logs in the bundled OpenObserve, or let an agent search them.
sidebar:
  order: 8
---

`nexul install` bundles [OpenObserve](https://openobserve.ai). The server sends every log line there, and errors from people's browsers too (marked `source=web`). Your stacks' container output is not here; read it on the stack's [Logs section](/docs/guide/stacks-and-deploys/#logs).

## Opening the logs UI

1. Open `/openobserve/` on your instance, such as `https://nexul.example.com/openobserve/`.
2. Sign in as `nexul@nexul.local` with the logs password. The installer printed it, and it is saved as `NEXUL_LOGS_PASSWORD` in the install directory's `.env`.
3. Look in the `nexul` stream of the `default` organization.

OpenObserve's own login guards this page, not Nexul's. You can change the password inside OpenObserve.

OpenObserve listens on localhost only and the server passes `/openobserve/` through to it, so there is no extra port to open.

## Searching logs from an agent

OpenObserve has its own MCP server. Point your MCP client at `https://<instance>/openobserve/api/default/mcp` (Streamable HTTP) with a basic `Authorization` header made from the logs email and password:

```sh
echo "Authorization: Basic $(printf '%s' "$NEXUL_LOGS_EMAIL:$NEXUL_LOGS_PASSWORD" | base64 | tr -d '\n')"
```

## The logs credentials

`nexul install` generates these into the install directory's `.env` and keeps them on every re-run:

| Variable | What it is |
| --- | --- |
| `NEXUL_LOGS_EMAIL` | The OpenObserve login email |
| `NEXUL_LOGS_PASSWORD` | The OpenObserve login password |
| `NEXUL_LOGS_TOKEN` | The token the server sends logs with |

OpenObserve reads them only on its first start. Don't edit them in `.env` afterwards, and keep the email as it is: the server sends logs as that user, with that token.

## A server run by hand

A server you start yourself has no OpenObserve. Send its logs to any OTLP/HTTP backend:

| Variable | What it sets |
| --- | --- |
| `NEXUL_OTLP_ENDPOINT` | The backend's base URL; `/v1/logs` is added |
| `NEXUL_OTLP_USER` | The basic auth user |
| `NEXUL_OTLP_TOKEN` | The basic auth token |

Without `NEXUL_OTLP_ENDPOINT`, logs go to stderr only.
