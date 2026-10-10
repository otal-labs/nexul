---
title: Paired computers
description: Connect your own computer running T3 Code so plays and @Agent mentions run there, and choose which computer and model each project uses.
sidebar:
  order: 9
---

Agent work in Nexul runs on your own computer, through T3 Code, with your permissions. Nobody else's turns ever run on your machine. A paired computer is not a [runner](/docs/guide/runners/): runners build and deploy, paired computers run agents.

## Add a computer

Adding a computer needs no Cloudflare and no domain: one command installs the Nexul app on the computer, and Nexul reaches T3 Code through it. The computer needs Linux or macOS, an account that can use `sudo`, and a way out to the internet; Windows is coming. On a Mac, T3 Code answers only while you are logged in at it (see [What the command installs](#what-the-command-installs)).

1. Open your settings, **Computers**, and press **Add a computer**.
2. Copy the command under **macOS / Linux** (**Windows** says Coming soon) and run it in a terminal on that computer, from your own account (not as root):

   ```sh
   curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token>
   ```

   The token in it is for this computer only and works once, within the hour the dialog names. `sudo` asks for your password once, to install a background service; everything else is installed for you, the account that typed `sudo`, and runs as you, never as root.
3. Watch the three checks. **Computer connected** passes once the Nexul app on the computer connects, **T3 Code found** once it finds T3 Code answering there, and **Paired** once Nexul has paired with T3 Code through the app's own connection. There's no pairing link to paste. A check that can't pass says what to do in its line: **Offline**, **Closed** (open the T3 Code desktop app), **Not running**, **Missing** (run the command again without `--no-t3`), or **Failed** with the reason.
4. Press **Set up**, then [set up the computer](/docs/guide/computer-setup/), and **Done**.

The computer is named after its hostname; **Rename** in its row's **…** menu changes that. If you close the dialog before running the command, the row reads **New computer** with **Add this computer again**, which makes a fresh command for the same row.

### What the command does on the computer

`sudo` is there only to place a system service. Everything else is installed for you, the person who typed `sudo`:

- the `nexul` command in your `~/.local/bin`, and the Nexul app with its credential in `~/.local/share/nexul` (`~/Library/Application Support/nexul` on a Mac), all owned by you, with the credential readable only by you;
- `nexul-computer`, a system service that runs the app as you, never as root: a systemd service on Linux, a LaunchDaemon in `/Library/LaunchDaemons` on a Mac. It starts at boot and keeps running after you log out.
- T3 Code, when it's missing; see [What the command installs](#what-the-command-installs).

Logged in as root, the command refuses: run it from your own account. On a computer without `sudo`, the shell says `sudo` is not found; as root, install sudo and add your account to the `sudo` group (`usermod -aG sudo <you>` on Debian and Ubuntu, the `wheel` group elsewhere), log in again, and rerun the command. Its logs are in `journalctl -u nexul-computer`, or `~/Library/Logs/nexul/nexul-computer.log` on a Mac. To remove it, remove the computer in Nexul, or run `nexul uninstall computer` from the same account: it tells Nexul, and a small root service the install left (a root LaunchDaemon on a Mac), which can only remove this app, takes the service and its files away; no sudo rule is involved. With sudo it removes them at once (`sudo ~/.local/bin/nexul uninstall computer`, since sudo does not search `~/.local/bin`). An agent can add a computer too, with [`computer_create`](/docs/guide/mcp-server/), which returns the same command.

### What the command installs

Before the Nexul app, the command looks for T3 Code in your account: a server already answering, the desktop app, or the `t3` command line. It reuses what it finds:

- **T3 Code answering**: left as it is. When T3 Code's own background service runs it, the command turns on lingering for you, so the service keeps running after you log out.
- **The desktop app, closed**: it says "Open T3 Code". Nexul pairs it once it runs.
- **The `t3` command line, not running**: it starts T3 Code's background service for you (`t3 service install`), with lingering on.

With none, it says what it's about to install, then installs T3 Code's command line with T3 Code's own installer into your `~/.local/bin`, as you and never as root, and runs it as your background service (`t3 service install`) on port 3773. It turns on lingering for you, so T3 Code starts at boot and keeps running after you log out, and installs `libatomic1`, which T3 Code needs and minimal server images leave out. It waits until T3 Code answers, then starts the Nexul app, and Nexul pairs T3 Code through it on its own: there's no pairing link to paste.

If T3 Code can't be installed or started, the command stops before the Nexul app and says why, and the same command works again. To leave T3 Code alone, add `--no-t3`:

```sh
curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token> --no-t3
```

For T3 Code's service on another port, add `--t3-port <port>` (Linux only: T3 Code's Mac service always uses 3773).

On a Mac, T3 Code's background service is one of your login items rather than a system service, so macOS runs it only while you are logged in at the Mac's screen; logging in over SSH does not start it. The Nexul app keeps the computer connected from boot, but until you log in Nexul can't reach T3 Code there, and runs aimed at that computer fail. With FileVault on, a Mac that restarts runs nothing at all, the app included, until someone logs in. A Mac that should take runs while nobody sits at it needs automatic login, which macOS allows only with FileVault off.

Once installed, the Nexul app checks T3 Code every 30 seconds. When T3 Code's background service stops answering for two checks in a row, the app restarts it (`t3 service restart`), at most once every 5 minutes, and tells Nexul it did. If your user service manager isn't running, so the service can't be restarted, the app reports that instead; `sudo loginctl enable-linger <you>` turns it on. On a Mac the restart works while you're logged in; logged out, there is nothing to restart into until you log in again. The desktop app is never restarted: when it's closed, open T3 Code.

## Keep it paired

Each computer is one row. A row shows a button only when something needs doing; everything else is in its **…** menu. Click the name to unfold the details.

A computer added with **Add a computer** reads each part on its own: **Online** or **Offline** (with when it was last seen) for the computer itself, then **T3 Code answering**, **Open T3 Code** when the desktop app is closed, **T3 Code not running**, or **Pairing failed** with the reason. Nexul pairs it again on its own before the 30-day session ends, whenever the app is connected. Its buttons are **Add this computer again** when the app is gone from it, **Re-pair now** when a pairing failed or ran out, and **Set up** or **Update skills** for its setup; the **…** menu holds **Set up** or **Re-run setup**, **Re-pair now**, **Rename** and **Remove**. Its details show T3 Code's state (**Running** with its port and version, **Not running**, or **Not installed**), then its facts.

**Add this computer again** shows when the computer has no Nexul app: the command was never run, or the app was removed because its owner's account was disabled. It opens the same dialog with a fresh command for the same row, so the computer keeps its name, project links and setup, and Nexul pairs it again once the app connects.

### Computers paired through a tunnel

A computer paired through a Cloudflare tunnel or by URL before **Add a computer** existed keeps working and keeps its old row: whether it's **Connected**, **Trying to connect**, or **Not connected**, its setup state, and **Pair**, **Re-pair**, **Set up** or **Update skills**. Its details show the address, the T3 Code version and the date the pairing lasts until. A pairing lasts 30 days, because T3 Code's session can't be refreshed: the row warns in its last days and shows **Re-pair**, which takes a fresh pairing link from T3 Code (in the desktop app, **Settings → Connections → Authorized clients → Create link**, then **Share** and **Copy link**; on the command line, `t3 pair`); once it has expired the row says it acts as unpaired. The tunnel keeps its hostname. New computers are only added with **Add a computer**.

### The computer's facts

A computer's Nexul app reports on it when it connects and every 6 hours: its hostname, OS and architecture, the app's version, T3 Code's state, port, version and how it is installed, `cloudflared`'s version when it is there, your git name and email, and the free disk space in your home folder. Nexul adds what T3 Code lists through the app: each provider's CLI version, whether it is signed in, its models, and T3 Code's projects with their folders. They are stored only when something changed, and **Facts changed** says when that was.

Facts are yours alone. Nobody else sees them, a workspace Owner included, and an agent reads them through `computer_list` only when it acts for you.

**Remove**, in the **…** menu, asks first, then deletes the pairing and revokes the computer's MCP token. On a computer added with **Add a computer**, it also revokes the Nexul app's credential: the app removes its `nexul-computer` service and files within a heartbeat, or the next time the computer comes online, and on the way out ends every T3 Code session labelled `Nexul`, so Nexul keeps no way into T3 Code there. A command made for the computer and never run stops working too. On a computer paired through a tunnel, Remove deletes the tunnel and hostname instead, which already makes it unreachable; Nexul's session inside T3 Code stays until it expires, or until you run `t3 auth session list` on the computer and `t3 auth session revoke <id>` on the `Nexul` entry. Re-pair never ends a session.

### The computer's MCP token

Setup gives the computer its own personal access token, "Nexul MCP on <computer>", for its providers to reach Nexul. It shows in the row's details. **Replace** mints a new one, shown once, and **Revoke MCP token** cuts it off. With none, press **Mint MCP token**.

## Choose where turns run

Two tabs in **Computers** decide which computer, T3 project, provider, and model a turn uses.

**Defaults** apply to `@Agent` in a channel or direct message, to `@Agent` and runs nobody presses (auto plays, the decisions check) in a project you haven't linked, and as the suggestion when a play asks where to run:

- **Default computer**, needed once you have more than one.
- **Fallback T3 project**, the T3 project to work in when no link applies.
- Provider, model, and model options. Leave them empty for the computer's or provider's default.
- **New threads start in**: **Project folder**, or **New worktree per thread** so runs side by side never edit the same files.

**Projects** lists every project you can open. Each row reads back what your turns there use, or **Uses your defaults**. Open one to pick a **Computer**, a **T3 project**, a model, and where new threads start, then **Save**. **Use my defaults** clears the link, and the next play you run there asks where again. Your link is yours alone: each teammate links the same project to their own computer.

A play never falls back to your defaults when you press it. The first run in a project you haven't linked asks where, and saves your answer here as the project's link; **Change** in the run dialog updates it. The dialog can also override the provider and model for one run; the link keeps its own model. The trail keeps what the run used, so changing your settings later never rewrites history.

## When a turn can't start

The page says why once, above or beside its play buttons, and each button repeats it as its tooltip. In chat, the message says why:

| Message | Fix |
|---|---|
| Add a computer in Settings to run plays. | Add a computer, or finish adding one. |
| Your computer's pairing has expired. Re-pair it in Settings. | Press **Re-pair now** (or **Re-pair** on a tunnel computer) on the computer's row. |
| Link this project in Settings → Computers → Projects, or set a fallback under Defaults. | Link the project, or set a **Fallback T3 project**. |
| Several computers are paired. Pick a default in Settings. | Set a **Default computer**. |
| T3 Code on your computer is offline. | Check the computer is **Online** and T3 Code is running on it. |
