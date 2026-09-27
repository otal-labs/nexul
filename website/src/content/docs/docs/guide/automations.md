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

Automations run on an **automations host**: a small service that starts a worker for each automation placed on it. `nexul install` installs one on the instance's own server, named `instance`, as the `nexul-automations-instance` service. It runs every Default automation out of the box, and new automations go to it unless you choose otherwise.

Each automation runs on exactly one host. To move one, pick another host on the automation's page, or set `host_id` with `automation_update` (null puts it back on `instance`). The host fetches its assignments from the instance with its own credential and gets each automation's worker a token scoped to that host: the instance accepts it only while the automation is placed there and the host is still enrolled. So moving an automation, or removing its host, stops it running on the old host, and two hosts never run the same automation.

An automation can also be deployed like any other stack, onto your own server through a runner, which gives it direct access to whatever else is running on that machine's network, or it can dial in with its own token from anywhere that can reach your instance, a laptop included.

### Adding an automations host

Add one from the automations hosts list in the web UI, or with the `host_create` MCP tool and `kind: "automations"`. Give it a name (lower case letters, digits and dashes); you get one install line for Linux or macOS and one for Windows, carrying a one-time enrollment code that works once and expires after an hour:

```sh
curl -fsSL https://nexul.io/automations.sh | NEXUL_VERSION=v0.2.1 sh -s -- --server <instance-url> --name worker-1 --code nxe_…
```

```powershell
$env:NEXUL_VERSION='v0.2.1'; & ([scriptblock]::Create((irm https://nexul.io/automations.ps1))) --server <instance-url> --name worker-1 --code nxe_…
```

Run it on the machine. It installs the host as the `nexul-automations-<name>` service, which trades the code for the host's own credential; it needs no Docker. On Linux the host runs as the `nexul` system user. A machine can run several automations hosts, each with its own name, directory and credential.

### Removing an automations host

**Remove** in the hosts list, or `host_delete` with `kind: "automations"`, revokes the host's credential and deletes it. A connected host uninstalls its own service; one that was offline does so when it next connects and is refused. Move its automations to another host before you remove it. On the machine itself, `nexul uninstall automations <name>` removes the service and tells the instance.

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

The SDK package's own commands, run from your automation's project directory. This is the SDK's `nexul`, installed with the package, not the `nexul` command that installs and upgrades an instance:

- `nexul init` — asks for your instance URL and a personal access token, writes `nexul.config.json`, and scaffolds `src/index.ts`.
- `nexul dev` — an interactive harness: pick one of the automation's registered topics, it fires a fixture event at your handler, and prints the outcome, captured logs, and every API call your handler *would* have made. No live effects — nothing actually reaches your instance.
- `nexul push <automation-id> [message]` — bundles your code and uploads it as a pending version.
- `nexul pull <automation-id> [version-id]` — fetches a version (the active one by default) back into `src/index.ts`.

### Testing

The `testing` module (`sdk/src/testing.ts`) exports `createMockContext`, the same mock context `nexul dev` uses under the hood: a context whose API client records every call instead of making it, so a plain unit test can assert on what your handler *would* have done without hitting a live server.

## Running one locally

1. In your automation's project directory, run `nexul init` and paste in a personal access token minted from **Settings → Tokens**.
2. Create the automation itself in the Nexul UI (**Automations → New automation**) to get its id and scopes.
3. Write handlers in `src/index.ts`, then use `nexul dev` to fire fixture events at them and check the logs and would-have-called API calls.
4. When it's ready, `nexul push <id>` uploads it as a pending version. The UI shows a diff against whatever's currently active; merging activates it and respawns the automation's worker.

## Versions

Every push creates an immutable version — code, who pushed it, when, and an optional message — landing as pending. The diff view against the active version is where you catch what you didn't mean to change. Rolling back means repointing at an older version, the same mechanism as any other merge.
