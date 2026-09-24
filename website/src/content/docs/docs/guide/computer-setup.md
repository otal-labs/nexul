---
title: Set up a computer
description: Get a paired computer ready for agent work from the setup wizard.
sidebar:
  order: 9
---

A provider can't run `@Agent` or a play on a paired computer until an agent
confirms that computer's setup. You never do this by hand: the setup wizard
connects Nexul's MCP server to each provider and installs the skills, then
each provider confirms itself.

Open **Settings → T3 pairing** and select **Set up** on the computer's row, or
continue from the last step of **Pair a computer**. A play refused because
setup isn't done shows a **Set up** button naming the computer in its run
dialog and its trail, which opens the same dialog straight away. Under
**Models**, pick the model each provider's setup turn runs on, from the models
T3 Code lists on that computer. Your default provider starts on your default
model and every other provider on its own default; **Provider default** leaves
the choice to the provider. Select **Start setup** and watch one row per
provider until it is confirmed; each row names the model its turn ran on.
Select **Retry** on a failed row to run only that provider again, on the model
picked for it.

The row shows **Setup confirmed** or **Needs setup**, with each provider's
confirmed-at time, and updates while setup runs. Select **Re-run setup** after
adding a provider or moving to a new machine. It re-checks and changes nothing
that already works.

For what each setup turn does, see
[Paired computers](/docs/guide/paired-computers/#set-up-a-computer).
