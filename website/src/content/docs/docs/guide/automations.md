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

The **Decisions check** isn't here: it's a [play](/docs/guide/plays/#the-decisions-check) with an auto play, switched on from the play.

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

   Return `true` for success. A thrown error, a timeout (30 seconds by default), or any other value counts as a failed run. `payload` is typed from the event catalog, and a list in it is always an array, `[]` when empty. `ctx` gives you `ctx.api`, a typed client acting with the automation's token, plus `ctx.config`, `ctx.secrets`, `ctx.log`, and `ctx.runPlay` ([Start a play](#start-a-play)).
4. Run `nexul dev`, pick a topic, and it fires a sample event at your handler. You see the outcome, the logs, and every API call it would have made. Nothing reaches your instance.
5. Run `nexul push <automation-id> "message"` to upload the code as a pending version.
6. On the automation's **Versions** tab, read the diff against the active code and press **Merge**. The automation restarts on the new code.

`nexul pull <automation-id>` fetches the active code back into `src/index.ts`. This `nexul` comes with the SDK; it's not the `nexul` that installs your instance.

For unit tests, `createMockContext` from `@nexul/sdk/testing` gives you the same recording context `nexul dev` uses. It records `ctx.runPlay` as the `POST /api/plays/queue` call it makes and answers with a queued run, unless your test scripts another answer under that key.

Two events mark a moment rather than one change. `ticket.unblocked` fires when a ticket's last open blocker reaches a done column, its blocked-by link is removed, or it's deleted, with `cause` saying which. `doc.settled` fires once a person's edits to a doc have stopped for ten minutes, with `first` true for a doc created inside that window; edits an agent makes through MCP, or an automation makes, never start the wait.

## Start a play

`ctx.runPlay` queues one of the workspace's ticket plays on a ticket, named by the play's label:

```ts
automation.on("ticket.test_failed", async (payload, ctx) => {
  const run = await ctx.runPlay("Fix with AI", payload.ticket.id, { priority: "high" });
  ctx.log("fix queued", { state: run.state, reason: run.reason });
  return true;
});
```

- The run waits in a person's queue exactly like an [auto play](/docs/guide/plays/#auto-plays)'s: on the ticket's developer, or its tester with `runOn: "tester"`, at `priority` `"high"`, `"normal"` (the default) or `"low"`. It counts toward the ticket's daily cap of automatic runs, and the play's show-when stage doesn't apply.
- It answers with `{ id, state, reason }`. `state` is `"queued"`, with `reason` `"paused"` when the ticket has reached its daily cap, or `"didnt_run"` with why, such as the ticket having no developer. Either way the ticket shows it.
- The automation's token needs the `plays:run` scope, and whoever created the automation must be able to run that play on that ticket. The default automations have no creator, so they can't start plays.

Play names are checked as you type. Run `nexul types <automation-id>` and it writes `src/plays.generated.d.ts` with the workspace's play names, so a typo, a doc play, or a play renamed since then is a type error in your editor. `nexul push` refreshes the file every time. A project made by `nexul init` already points `src/index.ts` at it; in an older project, add `/// <reference path="./plays.generated.d.ts" />` as the first line of `src/index.ts`. Labels are unique in a workspace, so a name is always one play.

Code you already pushed keeps the name it was written with. Renaming a play warns first: the play dialog lists the automations whose active or pending code still has the old label in quotes and asks before it saves, and `play_update` over MCP renames it and names them in its `warnings`. If someone renames or deletes that play anyway, `ctx.runPlay` throws, nothing is queued, and the run fails with the server's reason on the **Runs** tab, such as `no play named "Fix with AI" in this workspace`. Rename the play back, or update the name, push, and merge.

## Versions

Every push is kept as a version with its author, time, and message, and lands pending: a push never changes running code by itself. **Rollback** on an older version makes it active again.

## Secrets

Set secrets on the **Secrets** tab of the Automations page. Every automation in the workspace reads them as `ctx.secrets.NAME`; no other workspace sees them. Once saved, a value can be replaced (**Replace** on its row) or deleted but never read back.

## Where automations run

Automations run on an automations host. `nexul install` puts one on your instance's server, named `instance`, and new automations go there.

To run automations on another machine, for example to reach services on its network:

1. On the **Hosts** tab, press **Add automations host**.
2. Give it a **Host name** (lower-case letters, digits, and dashes) and the **Machine** it runs on. A host on a machine that exists takes **Manage** on that machine. A host on a machine Nexul doesn't know yet, or none, is the instance Owner's, since a host doesn't add machines.
3. Run the install line shown, for Linux or macOS, or for Windows, on that machine. It needs no Docker. The line carries a one-time code that expires after an hour.

A host belongs to its machine, like a runner. You see it when you can see its machine, and **Remove** takes **Manage** on the machine; a host on a machine you can't see is not found. Placing an automation on a host takes **Deploy to** on the host's machine as well as `automations:write` in the automation's workspace, and the instance host is open to every automation. Taking someone off a machine leaves the automations already placed there running; their next move is checked. Agents add and remove hosts with `host_create` and `host_delete` and `kind: "automations"`, and place an automation with `automation_update` and `host_id`.

Move an automation by picking another host on its page. It stops on the old host as it starts on the new one, so two hosts never run the same automation. To retire a host, move its automations off, then **Remove** it; it uninstalls itself. Removing a host moves its automations to the instance host and stops its credential. Adding a host again makes a new one: its automations stay on the instance host until you move them back.

## Switching off

Switch an automation off and it receives nothing until you switch it on again. Events in between are skipped, not replayed. A host that loses its connection is different: it picks up where it left off when it reconnects.

Its token stops working too: every API call with it is refused with `automation is switched off`. The one thing a switched-off automation's code can still do is connect and announce itself, so a new automation, which starts off, can register before you switch it on. It gets its configuration but no secrets and no events. Switch it back on and the same token works again; there's nothing to mint.
