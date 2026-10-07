---
title: Automations
description: Run your own TypeScript when something happens in a workspace, such as moving a ticket when its pull request merges.
sidebar:
  order: 12
---

An automation is code that runs when an event happens: a ticket is created, a pull request opens, a deploy fails. There's no rule builder. You write a handler in TypeScript against the Nexul SDK, push it, and Nexul runs it.

Each automation belongs to one workspace. It hears that workspace's events, plus instance-wide ones such as runners and DNS, and you switch and configure it on that workspace's **Automations** page. It never hears a direct message or a private channel.

## The defaults

Every workspace comes with its own copy of two, switched on:

- **Ticket finished** moves a ticket to a status you choose once all its pull requests have merged.
- **PR opened** moves a ticket to a status you choose when a linked pull request opens.

Open each one's **Configuration** tab and pick the status. Until you do, it skips every event and logs why on its **Runs** tab.

The **Decisions check** is listed with them. It's a [play](/docs/guide/plays/#the-decisions-check), not code, and starts off.

## Write your own

1. On the **Automations** page, press **New automation**. Name it and pick its **Scopes**: what its token may read and change, from the same [permissions](/docs/guide/api-and-tokens/) integrations use.
2. Add the SDK to an empty project. It's the `sdk/` package in the Nexul repository, not published to npm, so add it from a checkout; its commands need Bun. Then run `nexul init`. It asks for your instance URL and a personal access token from **Settings → Security → Tokens**, and writes `src/index.ts`.
3. Write your handlers:

   ```ts
   import { defineAutomation } from "@nexul/sdk/automation";

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

   Return `true` for success. A thrown error, a timeout (30 seconds by default), or any other value counts as a failed run. `ctx` gives you `ctx.api`, a typed client acting with the automation's token, plus `ctx.config`, `ctx.secrets`, and `ctx.log`.
4. Run `nexul dev`, pick a topic, and it fires a sample event at your handler. You see the outcome, the logs, and every API call it would have made. Nothing reaches your instance.
5. Run `nexul push <automation-id> "message"` to upload the code as a pending version.
6. On the automation's **Versions** tab, read the diff against the active code and press **Merge**. The automation restarts on the new code.

`nexul pull <automation-id>` fetches the active code back into `src/index.ts`. This `nexul` comes with the SDK; it's not the `nexul` that installs your instance.

For unit tests, `createMockContext` from `@nexul/sdk/testing` gives you the same recording context `nexul dev` uses.

## Versions

Every push is kept as a version with its author, time, and message, and lands pending: a push never changes running code by itself. **Rollback** on an older version makes it active again.

## Secrets

Set secrets on the **Secrets** tab of the Automations page. Every automation in the workspace reads them as `ctx.secrets.NAME`; no other workspace sees them. Once saved, a value can be replaced or deleted but never read back.

## Where automations run

Automations run on an automations host. `nexul install` puts one on your instance's server, named `instance`, and new automations go there.

To run automations on another machine, for example to reach services on its network:

1. On the **Hosts** tab, press **Add automations host**.
2. Give it a **Host name** (lower-case letters, digits, and dashes) and optionally a **Machine**.
3. Run the install line shown, for Linux or macOS, or for Windows, on that machine. It needs no Docker. The line carries a one-time code that expires after an hour.

Move an automation by picking another host on its page. It stops on the old host as it starts on the new one, so two hosts never run the same automation. To retire a host, move its automations off, then **Remove** it; it uninstalls itself.

## Switching off

Switch an automation off and it receives nothing until you switch it on again. Events in between are skipped, not replayed. A host that loses its connection is different: it picks up where it left off when it reconnects.
