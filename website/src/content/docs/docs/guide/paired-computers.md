---
title: Paired computers
description: Pair a user's T3 Code computer and choose where Agent turns run.
sidebar:
  order: 9
---

An Agent turn runs through the clicking user's own paired **Harness**. A
Runner builds and deploys stacks. It is a different product surface.

## Pair a computer

Open **Settings → T3 pairing** and select **Pair a computer**. The dialog has
three steps.

1. **Connect.** Name the computer and select **Create tunnel**. Nexul creates
   a tunnel for it on the instance's Cloudflare, with a hostname made from the
   name plus eight random characters that only the Nexul server can reach.
   Run the commands shown for your operating system on the computer; they
   install `cloudflared` as a background service. The dialog waits until the
   tunnel is online and T3 Code answers through it, then **Next** unlocks.
   The instance needs Cloudflare connected and Zero Trust enabled once; the
   dialog explains either missing piece with its fix.
2. **Pair T3 Code.** Run `t3 pair` on the computer and paste the one-time
   token it prints. The name and the tunnel hostname are already filled in.
   A refused token shows on the token field, so run `t3 pair` again for a
   fresh one. An unreachable T3 Code shows on the URL field.
3. **Set up.** Select **Start setup** to set up each provider on the
   computer; see [Set up a computer](#set-up-a-computer).

Until step 2 succeeds, the computer's row reads `pairing in progress`. Select
**Pair** on the row to finish pairing it without starting over.

### Pair by URL

For a machine the server can already reach, such as a VPS or a computer on
the same network, no tunnel is needed. On the **Connect** step open
**Advanced options** and select **Pair by URL**. Then enter:

- **Name**, such as `Home` or `VPS`.
- **T3 server URL**, the URL the server reaches that T3 Code instance at.
- **One-time pairing token**, copied from `t3 pair`.

### Sessions

The token is exchanged for a bearer session and the bearer is encrypted before
Nexul stores it. A pairing lasts 30 days because the upstream session has no
refresh flow. Select **Re-pair** before it expires, or when the row says
`expired · acts as unpaired`. A computer tunnel keeps its hostname when it is
re-paired. **Remove** deletes the pairing from Nexul, revokes the computer's MCP
token, and for a computer tunnel also deletes its tunnel, hostname, and Access
rule.

**MCP token** mints the computer its own personal access token, "Nexul MCP on
<computer>", for its providers' MCP configs. The token is shown once; minting
again replaces it, and **Revoke** on the row cuts it off. Un-confirming the
computer's setup revokes it too.

Each row reports the harness version and one presence state: **Connected**,
**Connecting**, or **Not connected**. A computer whose session is expired is
not usable even if its row remains.

## Set up a computer

A provider cannot run `@Agent` or a play on a computer until an agent confirms
its setup there. The dialog's **Set up** step runs that setup for you: one
turn per provider, one after another. Each turn connects Nexul's MCP server to
its provider with the computer's MCP token, installs the default skill set
(mattpocock/skills) and the nexul-memory skill into `~/.claude/skills/` and
`~/.agents/skills/`, and confirms the provider with the skills it discovered.
Each confirmed turn also confirms the computer, so one failed provider never
blocks the others. A failed provider can be retried on its own.

The step shows one row per provider: waiting, running with the agent's steps
folded under it, confirmed, or failed with **Retry**. Each computer row in
**Settings → T3 pairing** shows **Setup confirmed** or **Needs setup**, one line
per provider with its confirmed-at time, and **Set up** or **Re-run setup**,
which opens the dialog at this step. The row only shows the state; an agent
changes a confirmation through MCP and nowhere else.

Setup never overwrites an installed skill and keeps the token the providers
already hold, so re-running it on a confirmed computer only re-checks. Turns
run in the linked or fallback T3 project, else the first project T3 Code
lists, and their transcripts are kept with the token hidden.

## Choose defaults

The **Defaults** card in **Settings → T3 pairing** is used by `@Agent` in a
channel or direct message that has no project link. It has these fields:

- **Default computer**. Leave it empty when only one paired computer should be
  resolved automatically.
- **Fallback T3 project**. The project used when the chat context has no
  project.
- **Provider** and **Model**. Optional overrides. Empty values use the
  computer or provider default.

Several paired computers with no default produce the `no_default_computer`
readiness state. A paired computer without a T3 project produces
`no_default`.

## Link a project

Open a project's **Settings → T3 pairing** card to choose a **Computer**, a
**T3 project**, and optional **Provider** and **Model**. The project link wins
over the user's defaults. Clearing the link makes future Agent turns use the
user's defaults again.

The browser resolves a target before a play or chat mention starts. The UI
reports the reason when it cannot run:

- `unpaired`: pair a computer in Settings, or finish pairing one still in
  progress.
- `expired`: re-pair the expired computer.
- `no_harness_project`: select a project link or set a fallback project.
- `no_default_computer`: choose a default when more than one computer is
  paired.
- `offline`: the computer is not connected.

The play run dialog can override the resolved computer, provider, and model.
The selected values are saved on the trail so later settings changes do not
rewrite the run's history.

The pairing API is authenticated per user. Its routes include
`/api/pairing/computers`, `/api/pairing/computers/tunnel`,
`/api/pairing/computers/{id}/pair`, `/api/pairing/computers/{id}/setup/runs`,
`/api/pairing/computers/{id}/setup/providers/{provider}/retry`, `/api/pairing/defaults`,
`/api/pairing/projects/{id}`, and `/api/pairing/resolve`. A pairing failure
names the input it belongs to (`name`, `server_url`, or `token`) in the error
body's `errors` map.
