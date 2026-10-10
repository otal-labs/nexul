---
title: Set up a computer
description: Run setup once on a paired computer so its agents can reach Nexul and take on plays and chat mentions.
sidebar:
  order: 9
---

A paired computer can't run a play or an `@Agent` mention until each provider on it is set up. Setup is an agent turn per provider that connects it to Nexul's MCP server, installs the skills, and confirms the provider is ready. You don't edit any config by hand.

## Run setup

1. Open your settings, **Computers**, and press **Set up** on the computer's row (or **Re-run setup** in its **…** menu). **Set up**, the last step of **Add a computer**, opens the same screen.
2. Before the first run, **Options** is open. Switch off any provider you don't want set up.
3. Under **Models**, pick the model each provider's setup turn runs on, or leave **Provider default**.
4. Under **Folder**, pick which of T3 Code's projects the turns run in. Setup only writes files in your home folder, so any of them works; change it if the preselected folder no longer exists.
5. Press **Start setup**. One row per provider appears, and the running one opens to show the agent's steps. Click any row to open or fold its steps; each ends confirmed or failed.
6. A failed row opens on its reason. Press **Retry** on it to run only that provider again.

Once something has run, **Options** folds into one line that reads back the providers and folder; click it to change them. **Done** keeps your choices without running anything; **Cancel** drops them.

Each turn connects the provider to Nexul with the computer's own MCP token, installs the default skill set and the nexul-memory skill into `~/.claude/skills/` and `~/.agents/skills/`, and confirms the provider. One failed provider doesn't block the others.

A play stopped because setup isn't done shows a **Set up** button that opens the same screen.

## Check the state

The computer's row reads **Setup confirmed** with the providers it covers, **Needs setup**, **Setup failed** with the providers that failed, or the provider being set up right now. Unfold the row for a line per provider and when it was confirmed.

Press **Re-run setup**, in the row's **…** menu or on the setup screen, after adding a provider or moving to a new machine. It re-checks and changes nothing that already works: it never overwrites an installed skill and keeps the token the providers hold.

## Keep the skills current

A Nexul release sometimes brings a newer nexul-memory skill. A provider set up with an older one shows **skills out of date**, and a yellow dot appears on the settings gear and on **Computers**. Nothing stops working; the agent just follows the rules in its prompt until the skill is refreshed.

The row reads **Skills out of date**; press **Update skills** on it. One short turn rewrites the skills for every provider on the computer. Agents also check the skill's version once per session and refresh it themselves when it's behind.

For pairing, tunnels, and where turns run, see [Paired computers](/docs/guide/paired-computers/).
