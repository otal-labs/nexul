---
title: Paired computers
description: Connect your own computer running T3 Code so plays and @Agent mentions run there, and choose which computer and model each project uses.
sidebar:
  order: 9
---

Agent work in Nexul runs on your own computer, through T3 Code, with your permissions. Nobody else's turns ever run on your machine. A paired computer is not a [runner](/docs/guide/runners/): runners build and deploy, paired computers run agents.

## Pair a computer

The instance needs Cloudflare connected with Zero Trust enabled. T3 Code on the computer is optional: the tunnel command installs it when it's missing. If either is missing, the **Tunnel** step shows that in place of the form, with the one button that fixes it and **Try again**.

1. Open your settings, **T3 Code Setup → Computers**, and press **Pair a computer**.
2. On the **Tunnel** step, enter a **Computer name** and press **Create tunnel**. Change **T3 Code port** under **Advanced options** only if T3 Code doesn't run on its default port.
3. Pick **macOS / Linux** or **Windows** and run the command shown on that computer. It installs `cloudflared` if needed and keeps a tunnel open to the instance's Cloudflare account as a background service. The token in it is secret, so keep it to that computer. It then installs T3 Code if it's missing; add `--no-t3` (`-NoT3` on Windows) to skip that, and `--port <port>` (`-Port <port>`) if you changed **T3 Code port**.
4. Wait for both checks, **Tunnel online** and **T3 Code answering**, then press **Next**.
5. Pick how T3 Code is installed on the computer and make a pairing link there. **Desktop app** (the default) needs no terminal: in T3 Code open **Settings → Connections**, under **Authorized clients** choose **Create link** and **Create link** again, then on the new link choose **Share** and **Copy link**. **Command line** runs plain `t3 pair`, and **Not installed yet** shows the same command for the T3 Code the tunnel command installed; copy the **Pairing URL** it prints. Paste the link into **Pairing link**. Nexul reaches T3 Code over the tunnel, so only the token in the link is used; its address, a local one such as `192.168.1.107`, is ignored. A bare token still works. A refused link shows on the field: make a fresh one, because each works once.
6. On the last step, [set up the computer](/docs/guide/computer-setup/).

The computer gets a hostname made from its name plus eight random characters, and only the Nexul server can reach it. If you close the dialog early, the row reads **Pairing in progress**; press **Pair** on it to carry on.

### Pair by URL

For a machine the server can already reach, such as a VPS or a computer on the same network, skip the tunnel. On the **Tunnel** step choose **Pair by URL** under the form, and enter a **Name** and paste the **Pairing link**. The link's address fills the **T3 server URL**, which you can still edit. In T3 Code's desktop app, switch on **Network access** under **Settings → Connections** first, and pick **Local network** under **Reach this machine via** when you share the link; a link to `localhost` or `127.0.0.1` only works on that machine, so Nexul warns about it.

**Not installed yet** shows how to install T3 Code by hand. T3 Code's background service listens on `127.0.0.1` only, so on Linux the command writes a systemd override that sets `T3CODE_HOST=0.0.0.0` before `t3 service install`, and `t3 pair` then prints a link with the machine's first network address. That opens T3 Code's port on every interface, so keep it behind a firewall or a private network such as Tailscale. T3 Code's macOS service has no such setting: run `t3 serve --host <address>` there, or use a tunnel.

### Add a Linux computer with its runner

A Linux computer can also join through a runner of its own, which needs no Cloudflare. [`computer_create`](/docs/guide/mcp-server/) gives the command; run it on that computer from your own account:

```sh
curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token>
```

`sudo` is there only to place a system service. Everything else is installed for you, the person who typed `sudo`:

- the `nexul` command in your `~/.local/bin`, and the runner with its credential in `~/.local/share/nexul`, all owned by you, with the credential readable only by you;
- `nexul-computer`, a system service that runs the runner as you, never as root. It starts at boot and keeps running after you log out.
- T3 Code, when it's missing; see [What the command installs](#what-the-command-installs).

Logged in as root, the command refuses: run it from your own account. On a computer without `sudo`, the shell says `sudo` is not found; as root, install sudo and add your account to the `sudo` group (`usermod -aG sudo <you>` on Debian and Ubuntu, the `wheel` group elsewhere), log in again, and rerun the command. Its logs are in `journalctl -u nexul-computer`. To remove it, run `nexul uninstall computer` from the same account: it tells Nexul, and a small root service the install left, which can only remove this runner, takes the service and its files away; no sudo rule is involved. With sudo it removes them at once (`sudo ~/.local/bin/nexul uninstall computer`, since sudo does not search `~/.local/bin`).

### What the command installs

Before the runner, the command looks for T3 Code in your account: a server already answering, the desktop app, or the `t3` command line. It reuses what it finds:

- **T3 Code answering**: left as it is. When T3 Code's own background service runs it, the command turns on lingering for you, so the service keeps running after you log out.
- **The desktop app, closed**: it says "Open T3 Code". Nexul pairs it once it runs.
- **The `t3` command line, not running**: it starts T3 Code's background service for you (`t3 service install`), with lingering on.

With none, it says what it's about to install, then installs T3 Code's command line with T3 Code's own installer into your `~/.local/bin`, as you and never as root, and runs it as your background service (`t3 service install`) on port 3773. It turns on lingering for you, so T3 Code starts at boot and keeps running after you log out, and installs `libatomic1`, which T3 Code needs and minimal server images leave out. It waits until T3 Code answers, then starts the runner, and Nexul pairs T3 Code through it on its own: there's no pairing link to paste.

If T3 Code can't be installed or started, the command stops before the runner and says why, and the same command works again. To leave T3 Code alone, add `--no-t3`:

```sh
curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token> --no-t3
```

For T3 Code's service on another port, add `--t3-port <port>`.

## Keep it paired

Each computer is one row: its name, whether it's **Connected**, **Trying to connect**, or **Not connected**, and its setup state. A row shows a button only when something needs doing (**Pair**, **Re-pair**, **Set up**, **Update skills**); everything else is in its **…** menu. Click the name to unfold the details: the address, the T3 Code version, the date the pairing lasts until, each provider's setup, and the MCP token.

A pairing lasts 30 days, because T3 Code's session can't be refreshed. The row warns in its last days and shows **Re-pair**; once it has expired the row says it acts as unpaired. **Re-pair** is also in the **…** menu at any time; it takes a fresh pairing link the same way. The tunnel keeps its hostname.

**Remove**, in the **…** menu, asks first, then deletes the pairing and revokes the computer's MCP token. On a computer added with the command, it also revokes the runner's credential: the runner removes its `nexul-computer` service and files within a heartbeat, or the next time the computer comes online, and on the way out ends every T3 Code session labelled `Nexul`, so Nexul keeps no way into T3 Code there. A command made for the computer and never run stops working too. On a computer paired through a tunnel, Remove deletes the tunnel and hostname instead, which already makes it unreachable; Nexul's session inside T3 Code stays until it expires, or until you run `t3 auth session list` on the computer and `t3 auth session revoke <id>` on the `Nexul` entry. Re-pair never ends a session.

### The computer's MCP token

Setup gives the computer its own personal access token, "Nexul MCP on <computer>", for its providers to reach Nexul. It shows in the row's details. **Replace** mints a new one, shown once, and **Revoke MCP token** cuts it off. With none, press **Mint MCP token**.

## Choose where turns run

Two tabs in **T3 Code Setup** decide which computer, T3 project, provider, and model a turn uses.

**Defaults** apply to `@Agent` in a channel or direct message, to `@Agent` and runs nobody presses (auto plays, the decisions check) in a project you haven't linked, and as the suggestion when a play asks where to run:

- **Default computer**, needed once you pair more than one.
- **Fallback T3 project**, the T3 project to work in when no link applies.
- Provider, model, and model options. Leave them empty for the computer's or provider's default.
- **New threads start in**: **Project folder**, or **New worktree per thread** so runs side by side never edit the same files.

**Projects** lists every project you can open. Each row reads back what your turns there use, or **Uses your defaults**. Open one to pick a **Computer**, a **T3 project**, a model, and where new threads start, then **Save**. **Use my defaults** clears the link, and the next play you run there asks where again. Your link is yours alone: each teammate links the same project to their own computer.

A play never falls back to your defaults when you press it. The first run in a project you haven't linked asks where, and saves your answer here as the project's link; **Change** in the run dialog updates it. The dialog can also override the provider and model for one run; the link keeps its own model. The trail keeps what the run used, so changing your settings later never rewrites history.

## When a turn can't start

The page says why once, above or beside its play buttons, and each button repeats it as its tooltip. In chat, the message says why:

| Message | Fix |
|---|---|
| Pair a computer in Settings to run plays. | Pair a computer, or finish one in progress. |
| Your computer's pairing has expired. Re-pair it in Settings. | Press **Re-pair** on the computer's row. |
| Link this project in Settings → T3 Code Setup → Projects, or set a fallback under Defaults. | Link the project, or set a **Fallback T3 project**. |
| Several computers are paired. Pick a default in Settings. | Set a **Default computer**. |
| T3 Code on your computer is offline. | Start T3 Code, and check the tunnel is up. |
