---
title: Automations
description: First-party event-driven code that reacts to what happens in your workspace, and how to write one.
sidebar:
  order: 12
---

An automation is first-party code: when an event happens in your workspace, a function runs. There's no rule builder or condition DSL — customization means writing code against the Nexul SDK.

## The model

- **Default automations** ship with the instance, written on the same public SDK as anything you'd write yourself — standing proof the SDK works. **Custom automations** are yours.
- Every automation declares, in code, a name, a description, the events it subscribes to (`on('topic', handler)`), and any config knobs it needs. The platform only displays what's declared — it never edits it.
- Automations **dial in**: the SDK opens one outbound connection to your instance, authenticated by the automation's own token, and events stream down that connection as they happen. There's no inbound webhook URL and no signature verification to manage for first-party code — that's the model signed webhooks use for third-party integrations instead. A reconnect resumes from the last event the automation saw, so downtime doesn't lose anything.
- A handler is an async function for one subscribed event. Returning `true` means success; an exception, a timeout, or anything else counts as a failed run. Runs default to a 30 second timeout and report their outcome (with captured logs) back over the dial-in connection.

## Where automations run

The **automations host** is a small container bundled with your instance from install — it runs every Default automation out of the box, plus any small Custom ones you write. Bigger Custom automations can instead be deployed like any other stack, onto your own server through a runner, which gives them direct access to whatever else is running on that machine's network. An automation can also just dial in from anywhere that can reach your instance — a laptop included.

## Tokens and secrets

Each automation gets a scoped token minted when you create it; you pick its scopes from the same [permission vocabulary](/docs/guide/api-and-tokens/) integrations use. The token is what gates what the automation's code can touch through the API — including changing Nexul itself. Revoking it kills access immediately.

Secrets are a shared workspace pool, GitHub-Actions-style: set a name and value once in settings, and every automation can read it as `ctx.secrets.NAME`. Values are write-only after saving — names stay visible, values never do.

## The SDK

The SDK (`sdk/`) is one package used by both automations and integrations: a typed API client, typed event payloads, and an automations entry point.

```ts
import { defineAutomation } from "@nexul/sdk/automation";
import { DialinClient } from "@nexul/sdk/client";

const automation = defineAutomation({
  name: "my-automation",
  description: "Describe what this automation does.",
  config: {},
});

automation.on("ticket.created", async (payload, ctx) => {
  ctx.log("ticket created", { id: payload.ticket.id });
  return true;
});

export default automation;
```

`ctx` gives a handler `ctx.api` (the typed client), `ctx.config`, `ctx.secrets`, and `ctx.log`.

### The `nexul` CLI

The package's own commands, run from your automation's project directory:

- `nexul init` — asks for your instance URL and a personal access token, writes `nexul.config.json`, and scaffolds `src/index.ts`.
- `nexul dev` — an interactive harness: pick one of the automation's registered topics, it fires a fixture event at your handler, and prints the outcome, captured logs, and every API call your handler *would* have made. No live effects — nothing actually reaches your instance.
- `nexul push <automation-id> [message]` — bundles your code and uploads it as a pending version.
- `nexul pull <automation-id> [version-id]` — fetches a version (the active one by default) back into `src/index.ts`.

### Testing

The `testing` module (`sdk/src/testing.ts`) exports `createMockContext`, the same mock context `nexul dev` uses under the hood: a context whose API client records every call instead of making it, so a plain unit test can assert on what your handler *would* have done without hitting a live server.

## Running one locally

1. In your automation's project directory, run `nexul init` and paste in a personal access token minted from **Settings → Personal access tokens**.
2. Create the automation itself in the Nexul UI (**Automations → New automation**) to get its id and scopes.
3. Write handlers in `src/index.ts`, then use `nexul dev` to fire fixture events at them and check the logs and would-have-called API calls.
4. When it's ready, `nexul push <id>` uploads it as a pending version. The UI shows a diff against whatever's currently active; merging activates it and respawns the automation's worker.

## Versions

Every push creates an immutable version — code, who pushed it, when, and an optional message — landing as pending. The diff view against the active version is where you catch what you didn't mean to change. Rolling back means repointing at an older version, the same mechanism as any other merge.
