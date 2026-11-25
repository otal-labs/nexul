---
title: Audit log
description: What the audit log records, how to keep it up to three years, and how to stream each change, signed, to your own log store.
sidebar:
  order: 4
---

The **audit log** is the instance's record of changes: one row each time someone, or an agent acting for them, changes something. Every instance keeps it 45 days and serves it at `GET /api/audit` to whoever holds **Read audit log**. With an [enterprise license](/docs/guide/enterprise-license/) key that includes **Audit log**, you can keep it longer and send each row to your own log store as it is written.

## What is recorded

One row per change:

- each request through the API that changes something, from the browser, an integration, an automation or a script;
- each call an agent makes over [MCP](/docs/guide/mcp-server/) to a tool that changes something;
- each message a [bot](/docs/guide/bots/) posts.

A row names who made the change (a person, an integration, an automation or a bot), the token it came with, the action (`PATCH /api/tickets/…` or `mcp ticket_update`) and when. It holds no request body.

Not recorded: reads (opening a page, listing tickets, a `_get` tool), and what people do on their own computers, which is [computer activity](/docs/guide/paired-computers/) and has its own permission.

## Keep it longer

In **Settings → Instance → Audit log**, set **Days kept** to any whole number from 45 to 1,095 and click **Save**. Each day the rows older than that are deleted. The published events and deploy logs Nexul keeps for itself stay at 45 days.

If the key lapses, the days you saved still hold: nothing the longer setting was keeping is deleted. To go back to 45 days, click **Remove settings**, which works with or without a key. The next daily pass then deletes the older rows.

## Stream it

Each row can be posted to one HTTPS address as it is written: Splunk's HTTP event collector, Datadog, Elastic, a Loki gateway or your own function.

1. Enter the **Destination**, an `https://` address. `http://` works only for `localhost`, for trying a receiver out.
2. Leave **Signing secret** blank to have one made, or paste your own. A [secret reference](/docs/guide/external-secrets/) such as `op://vault/item/field` is read from your secrets manager.
3. Click **Send a test**. One signed sample row goes to the destination, and the card shows what it answered. Nothing is saved or queued.
4. Turn on **Send each change** and click **Save**. A secret that Nexul made shows once under the field. Copy it into your receiver now.

Only rows written while the stream is on are sent. Rows written before you turned it on are not.

### What arrives

One `POST` per row, with `Content-Type: application/json`:

```json
{
  "delivery_id": "01a214cc-9f0e-7c4b-a1d2-5e3f60718293",
  "id": "01a214cc-45ad-79d3-852a-16fb42261530",
  "created_at": "2026-11-14T09:30:00Z",
  "actor_type": "user",
  "actor_id": "01a214cb-84a2-7026-b70a-12dc1b02b9a2",
  "token_id": "",
  "action": "PATCH /api/projects/abc",
  "instance_url": "https://nexul.example.com"
}
```

`X-Nexul-Signature` carries `sha256=` and the hex HMAC-SHA256 of the raw body under the signing secret, the same header [integrations](/docs/guide/api-and-tokens/) receive. Check it before you trust the body:

```js
import { createHmac, timingSafeEqual } from "node:crypto";

const expected = Buffer.from("sha256=" + createHmac("sha256", secret).update(rawBody).digest("hex"));
const given = Buffer.from(request.headers["x-nexul-signature"] ?? "");
const valid = given.length === expected.length && timingSafeEqual(given, expected);
```

### Retries and duplicates

A delivery counts as delivered only when the receiver answers with a 2xx. A redirect is not followed. Any other answer, or none within 15 seconds, is retried after 1, 2, 4, 8 and 16 seconds, then every 30. After 8 attempts the row stops and shows under **Deliveries** as failed, with the reason. Fix the receiver, then click **Retry failed** to send those rows again.

Delivery is at least once. A row can arrive twice, for example if the server restarts between your 2xx and recording it. A retry can also arrive after newer rows. Dedupe on `id`, which is the same on every attempt, and order by `created_at`. `delivery_id` is new on each attempt.

Rows wait in a queue as long as the log keeps them. A receiver that is down for a day misses nothing: the rows queue up, and the card shows how many are waiting and when the last one went out. Turning the stream off keeps what is queued, and it is sent when you turn the stream back on. While the key is lapsed, nothing new is queued and nothing is sent.

**Rotate secret** makes a new signing secret and shows it once. Deliveries carry the new signature from then on.

## From an agent

`instance_get` returns the audit log settings with the rest of the instance's state: the days kept, the destination, whether the stream is on, and the waiting and failed counts. It never returns the secret. `instance_update` with an `audit` object sets `retention_days`, `stream_url`, `stream_secret` and `stream_enabled`. `retry_failed` sends the failed rows again, and `remove` deletes the settings. Reading the log itself is `GET /api/audit`.
