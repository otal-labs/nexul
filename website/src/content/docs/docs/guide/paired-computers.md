---
title: Paired computers
description: Connect your own computer running T3 Code so plays and @Agent mentions run there, and choose which computer and model each project uses.
sidebar:
  order: 9
---

Agent work in Nexul runs on your own computer, through T3 Code, with your permissions. Nobody else's turns ever run on your machine. A paired computer is not a [runner](/docs/guide/runners/): runners build and deploy, paired computers run agents.

## Pair a computer

You need T3 Code running on the computer, and the instance needs Cloudflare connected with Zero Trust enabled. The dialog tells you if either is missing and how to fix it.

1. Open your settings, **T3 Code Setup → Computers**, and press **Pair a computer**.
2. Enter a **Computer name** and press **Create tunnel**. Change **T3 Code port** under **Advanced options** only if T3 Code doesn't run on its default port.
3. Run the command shown for your system on that computer. It installs `cloudflared` if needed and keeps a tunnel open to the instance's Cloudflare account as a background service. The token in it is secret, so keep it to that computer.
4. Wait for both checks, **Tunnel online** and **T3 Code answering**, then press **Next**.
5. Run `t3 pair` on the computer and paste the one-time token it prints. A refused token shows on the token field: run `t3 pair` again for a fresh one.
6. On the last step, [set up the computer](/docs/guide/computer-setup/).

The computer gets a hostname made from its name plus eight random characters, and only the Nexul server can reach it. If you close the dialog early, the row reads `pairing in progress`; press **Pair** on it to carry on.

### Pair by URL

For a machine the server can already reach, such as a VPS or a computer on the same network, skip the tunnel. On the first step open **Advanced options**, choose **Pair by URL**, and enter a **Name**, the **T3 server URL** the server reaches it at, and the **One-time pairing token** from `t3 pair`.

## Keep it paired

Each row shows the T3 Code version and a dot: green when connected, gray when not. Hover the dot to see whether Nexul is still trying to connect.

A pairing lasts 30 days, because T3 Code's session can't be refreshed. Press **Re-pair** before then, or when the row says it has expired and acts as unpaired. The tunnel keeps its hostname.

**Remove** deletes the pairing, revokes the computer's MCP token, and deletes its tunnel and hostname. Neither Remove nor Re-pair ends Nexul's session inside T3 Code, which offers no way to do that from outside. To end it before it expires, run `t3 auth session list` on the computer and `t3 auth session revoke <id>` on the `Nexul` entry. A removed tunnel computer is already unreachable.

### The computer's MCP token

Setup gives the computer its own personal access token, "Nexul MCP on <computer>", for its providers to reach Nexul. It shows on the row. **Replace** mints a new one, shown once, and **Revoke MCP token** cuts it off. With none, press **Mint MCP token**.

## Choose where turns run

Two tabs in **T3 Code Setup** decide which computer, T3 project, provider, and model a turn uses.

**Defaults** apply to `@Agent` in a channel or direct message, and to every project you haven't linked:

- **Default computer**, needed once you pair more than one.
- **Fallback T3 project**, the T3 project to work in when no link applies.
- Provider, model, and model options. Leave them empty for the computer's or provider's default.
- **New threads start in**: **Project folder**, or **New worktree per thread** so runs side by side never edit the same files.

**Projects** lists every project you can open. Each row reads back what your turns there use, or **Uses your defaults**. Open one to pick a **Computer**, a **T3 project**, a model, and where new threads start, then **Save**. **Use my defaults** clears the link. Your link is yours alone: each teammate links the same project to their own computer.

A play's run dialog can override the computer, provider, and model for one run. The trail keeps what the run used, so changing your settings later never rewrites history.

## When a turn can't start

The page says why once, above or beside its play buttons, and each button repeats it as its tooltip. In chat, the message says why:

| Message | Fix |
|---|---|
| Pair a computer in Settings to run plays. | Pair a computer, or finish one in progress. |
| Your computer's pairing has expired. Re-pair it in Settings. | Press **Re-pair** on the computer. |
| Link this project in Settings → T3 Code Setup → Projects, or set a fallback under Defaults. | Link the project, or set a **Fallback T3 project**. |
| Several computers are paired. Pick a default in Settings. | Set a **Default computer**. |
| T3 Code on your computer is offline. | Start T3 Code, and check the tunnel is up. |
