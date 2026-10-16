---
title: Triage a bug in Nexul
description: Ask your own agent to diagnose a bug in Nexul itself and file it as an issue, with nothing private in it.
sidebar:
  order: 9
---

When Nexul itself misbehaves, your agent can turn it into a diagnosed issue on [otal-labs/nexul](https://github.com/otal-labs/nexul). This is for bugs in Nexul. A problem in your own app belongs on your own board.

Every [computer's setup](/docs/guide/computer-setup/) installs the `nexul-triage` skill beside `nexul-memory`, and agents keep it current the same way. Nothing starts it on its own: in a session on a set-up computer, say "triage this with Nexul" and describe what went wrong.

## What the agent does

1. Gathers the facts through Nexul's MCP tools: the server version, each runner's version, OS and architecture, and the `trace_id` of the failing request if you have it. It asks you for the install type.
2. Reads the instance's recent errors with [`instance_get` and its `logs` filter](/docs/guide/logs/#reading-the-logs-through-nexul), if you hold the **Logs** area. Without it, it asks what you saw.
3. Clones the source at your installed version, searches open and closed issues for a duplicate, and checks whether a newer release already fixes it.
4. Goes as far as a cause and a suspected file, and a failing test if that is cheap. It never patches your installed Nexul, and opens a fix pull request only if you ask.

## What a report may carry

The versions, OS, architecture and install type, the error and stack trace with tokens redacted, and the diagnosis. A few quoted log lines only if you hold Logs.

Never your instance's address or any host name, and never the names of workspaces, projects, stacks or people. The agent replaces them with placeholders such as `example.com`, `acme` and `alice` before it drafts.

## Approve, then it files

The agent shows you the full draft and posts nothing until you say yes to that draft; a changed draft needs a new yes. It files through your own GitHub (`gh` or the GitHub MCP), so you are the author, using the **Triage report** form and its `via-triage` label. The form's last field reads like `alice via Nexul triage · <agent> (<model>) · Nexul 0.3.40`; the name is the one you give, and you can edit it in the draft.

Without a GitHub account, the agent gives you a prefilled `issues/new?template=triage_report.yml` link and the draft as text, to hand to someone who has one.
