---
title: Paired computers
description: Pair a user's T3 Code computer and choose where Agent turns run.
sidebar:
  order: 9
---

An Agent turn runs through the clicking user's own paired **Harness**. A
Runner builds and deploys stacks. It is a different product surface.

## Pair a computer

Open **Settings → T3 pairing** and select **Pair a computer**. On the machine
running T3 Code, run `t3 pair`, then enter these values:

- **Name**, such as `Home` or `VPS`.
- **T3 server URL**, the URL printed or configured for that T3 Code instance.
- **One-time pairing token**, copied from `t3 pair`.

The token is exchanged for a bearer session and the bearer is encrypted before
Nexul stores it. A pairing lasts 30 days because the upstream session has no
refresh flow. Select **Re-pair** before it expires, or when the row says
`expired · acts as unpaired`. **Remove** deletes the pairing from Nexul.

Each row reports the harness version and one presence state: **Connected**,
**Connecting**, or **Not connected**. A computer whose session is expired is
not usable even if its row remains.

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

- `unpaired`: pair a computer in Settings.
- `expired`: re-pair the expired computer.
- `no_harness_project`: select a project link or set a fallback project.
- `no_default_computer`: choose a default when more than one computer is
  paired.
- `offline`: the computer is not connected.

The play run dialog can override the resolved computer, provider, and model.
The selected values are saved on the trail so later settings changes do not
rewrite the run's history.

The pairing API is authenticated per user. Its routes include
`/api/pairing/computers`, `/api/pairing/defaults`,
`/api/pairing/projects/{id}`, and `/api/pairing/resolve`.
