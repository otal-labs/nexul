---
title: Set up a computer
description: Run setup once on a paired computer so its agents can reach Nexul and take on plays and chat mentions.
sidebar:
  order: 9
---

A paired computer can't run a play or an `@Agent` mention until each provider on it is set up. Setup is an agent turn per provider that connects it to Nexul's MCP server, installs the skills, and confirms the provider is ready. You don't edit any config by hand.

## Run setup

1. Open your settings, **Computers**, and press **Set up** on the computer's row (or **Re-run setup** in its **…** menu). **Set up**, the last step of **Add a computer**, opens the same screen.
2. Each provider T3 Code lists on the computer has a row with a switch. Switch off any provider you don't use. The switch saves straight away, so a provider you switch off stops showing in the model pickers for plays and chat.
3. Pick the model each provider's setup turn runs on, beside its name, or leave **Provider default**.
4. Under the list, **Runs in** picks which of T3 Code's projects the turns run in. Setup only writes files in your home folder, so any of them works; change it if the preselected folder no longer exists. A computer with no T3 project has nothing to pick: setup refuses and says to create one, with **Create empty project** under Settings → Computers → Projects or `computer_project_create` with a `name`.
5. Press **Set up all** to set up every switched-on provider at once, or **Set up** on one row to run just that provider. Providers set up side by side, and you can start another row while one is still running.
6. Click a provider's name to open or fold the agent's steps; the running one opens by itself. Each ends confirmed or failed. A failed row opens on its reason; press **Retry** on it to run only that provider again.

**Done** keeps your model and folder choices without running anything; **Cancel** drops them. The switches are already saved.

Each turn connects the provider to Nexul with the computer's own MCP token, installs the default skill set and the nexul-memory and [nexul-triage](/docs/guide/triage/) skills into `~/.claude/skills/` and `~/.agents/skills/`, and confirms the provider. One failed provider doesn't block the others.

## Switch a provider off

A switched-off provider is left out of every model picker, and a play or `@Agent` mention aimed at it is refused with a note to switch it on. When a play or mention names no provider, the computer's first switched-on, confirmed provider runs it. A provider that's on but not set up yet still shows in the pickers, greyed out with **Setup required**, so you can see what setup would add.

Switching a provider back on keeps its earlier confirmation, so it doesn't need setting up again.

A play stopped because setup isn't done shows a **Set up** button that opens the same screen.

## Check the state

The computer's row reads **Setup confirmed** with the providers it covers, **Needs setup**, **Setup failed** with the providers that failed, or the provider being set up right now. Unfold the row for a line per provider and when it was confirmed.

Press **Re-run setup** in the row's **…** menu, or **Run all again** on the setup screen, after adding a provider or moving to a new machine. It re-checks and changes nothing that already works: it never overwrites an installed skill and keeps the token the providers hold.

## Keep the skills current

A Nexul release sometimes brings a newer Nexul skill. A provider set up with an older one shows **skills out of date**, and a yellow dot appears on the settings gear and on **Computers**. Nothing stops working; the agent just follows the rules in its prompt until the skill is refreshed.

The row reads **Skills out of date**; press **Update skills** on it. One short turn rewrites the skills for every provider on the computer. Agents also check the skill's version once per session and refresh it themselves when it's behind.

For pairing and where turns run, see [Paired computers](/docs/guide/paired-computers/).
