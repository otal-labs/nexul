---
title: Logs
description: Read the instance's own logs through Nexul, from an agent or the API, or open the bundled OpenObserve yourself.
sidebar:
  order: 8
---

`nexul install` bundles [OpenObserve](https://openobserve.ai). Four sources write to it. Your stacks' container output is not here; read it on the stack's [Logs section](/docs/guide/stacks-and-deploys/#logs).

## Where the lines come from

| `source` | What it is |
| --- | --- |
| `server` | Every line the server logs. |
| `web` | Errors and warnings from people's browsers. |
| `runner` | The log of each runner that serves a [machine](/docs/guide/runners/), the one bundled with the instance included. |
| `automations` | The log of each [automations host](/docs/guide/automations/), the one bundled with the instance included. A script's own output is not here; it stays with the automation's runs. |

A runner or host sends its lines to the server over the connection it already holds, and the server writes them to OpenObserve. Nothing needs to reach OpenObserve from another machine, and no OpenObserve login leaves the server. Each line is stamped by the server with the `source`, the runner's or host's `name`, its `machine` and its `version`, taken from the credential it connected with, so a line cannot claim to come from somewhere else: an attribute with one of those names is dropped. Lines go out every 2 seconds with tokens redacted, a runner's at its configured log level (`info` by default). A runner's or host's own journal keeps them too.

While the server is unreachable a runner or host keeps its newest 1000 lines and sends them when it connects again, so the line saying why the connection dropped is there. If more than 1000 pile up, the oldest go and the next batch carries one `dropped N log lines` warning. Restarting the runner or host loses what was still queued; its journal still has it.

A [paired computer](/docs/guide/paired-computers/)'s runner never sends its lines. They stay on that computer, and its owner, or anyone its share gives **Read logs**, reads them through Nexul instead.

## Reading the logs through Nexul

Call `instance_get` with a `logs` object. Every field is optional:

| Field | What it does |
| --- | --- |
| `source` | Only `server`, `web`, `runner` or `automations`. All of them when left out. |
| `level` | The lowest level returned: `debug`, `info`, `warn` or `error`. Defaults to `info`. |
| `since`, `until` | The time range, as a duration back such as `30m` or `2d`, or a time such as `2026-09-25T10:00:00Z`. Defaults to the last hour, and can span at most 7 days. |
| `trace_id` | Only the lines of one request, from a `trace_id` another line carries. |
| `name` | Only the lines of the runner or automations host with this name. |
| `text` | Only lines whose message contains this text, ignoring case. It cannot hold a backslash. |
| `limit`, `offset` | 1 to 500 lines, 100 by default, and how many to skip. |

Lines come back newest first, in the same `items`, `total`, `has_more` and `next_offset` shape as every list. Each line has its `time`, `level`, `source`, `message` and `trace_id`. A runner's or host's lines add `name`, `machine` and `version`, a browser's add `user_id` and `url`, and every other field the line carries sits in `attrs`: the `error` and ids a server line logs are there.

The answer also carries the `until` it read up to. A first page pins `until` to now; pass it back with `next_offset` for the next page, so lines logged meanwhile do not shift the pages.

The same read is `GET /api/instance/logs`, with those fields as query strings, for the browser, an integration token with `logs:read`, or `curl`.

Before a line leaves the server, any field whose name holds `token`, `secret`, `password`, `key`, `auth`, `cookie`, `credential` or `env` is dropped, every `dep_` or `ses_` token and `token=` value is replaced by `[redacted token]`, and a message or field value is cut at 2000 characters.

When the instance has no log store (a server run by hand), the call is refused with "this instance keeps no logs". When OpenObserve does not answer, it fails with a conflict naming the log store and telling you to retry, never an empty list.

An agent can use these reads to [triage a bug in Nexul](/docs/guide/triage/) and file it as an issue.

## Who can read them

Reading takes `logs:read`, the **Logs** instance area. The instance Owner holds it. Nobody else does until the Owner hands it out: in **Settings**, open an instance entry's **Access** card, choose **The instance's logs**, and allow **Read the instance's logs** for a role or a person. `instance_get` itself still needs `instance:read`, so a role that should read logs needs both. See [People and access](/docs/guide/people-and-access/#instance-permissions) for how instance areas work.

Holding Logs does not open OpenObserve. Whoever holds it sees exactly what the filter above lets out, and nothing the Owner's password could reach.

## Opening OpenObserve yourself

The Owner can still go to OpenObserve directly, with the logs password.

1. Open `/openobserve/` on your instance, such as `https://nexul.example.com/openobserve/`.
2. Sign in as `nexul@nexul.local` with the logs password. The installer printed it, and it is saved as `NEXUL_LOGS_PASSWORD` in the install directory's `.env`.
3. Look in the `nexul` stream of the `default` organization.

OpenObserve's own login guards this page, not Nexul's, and the password sees every stream and setting. OpenObserve listens on localhost only and the server passes `/openobserve/` through to it, so there is no extra port to open.

OpenObserve has its own MCP server too. Point your MCP client at `https://<instance>/openobserve/api/default/mcp` (Streamable HTTP) with a basic `Authorization` header made from the logs email and password:

```sh
echo "Authorization: Basic $(printf '%s' "$NEXUL_LOGS_EMAIL:$NEXUL_LOGS_PASSWORD" | base64 | tr -d '\n')"
```

That road knows nothing of Nexul's people or its redaction, so keep it for yourself and give agents `instance_get`.

## The logs credentials

`nexul install` generates these into the install directory's `.env` and keeps them on every re-run:

| Variable | What it is |
| --- | --- |
| `NEXUL_LOGS_EMAIL` | The OpenObserve login email |
| `NEXUL_LOGS_PASSWORD` | The OpenObserve login password |
| `NEXUL_LOGS_TOKEN` | The token the server sends logs with |

OpenObserve reads them only on its first start. Don't edit them in `.env` afterwards, and keep the email as it is: the server sends logs as that user, with that token, and reads them back with that email and password. `nexul upgrade` hands the server the email and password, so an install made before logs could be read gets them on its next upgrade. Changing the password inside OpenObserve makes the server's reads fail until `.env` and the server's service match it again.

## A server run by hand

A server you start yourself has no OpenObserve. Send its logs to any OTLP/HTTP backend:

| Variable | What it sets |
| --- | --- |
| `NEXUL_OTLP_ENDPOINT` | The backend's base URL; `/v1/logs` is added |
| `NEXUL_OTLP_USER` | The basic auth user |
| `NEXUL_OTLP_TOKEN` | The basic auth token |

Without `NEXUL_OTLP_ENDPOINT`, logs go to stderr only, and reading them through Nexul is refused.
