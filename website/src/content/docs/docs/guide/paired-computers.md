---
title: Paired computers
description: Connect your own computer running T3 Code so plays and @Agent mentions run there, and choose which computer and model each project uses.
sidebar:
  order: 9
---

Agent work in Nexul runs on your own computer, through T3 Code, with your permissions. Nobody else sees your computer or runs anything on it unless you [share it](#share-a-computer) with them by name. A paired computer is not a [runner](/docs/guide/runners/): runners build and deploy, paired computers run agents.

## Add a computer

Adding a computer needs no Cloudflare and no domain: one command installs the Nexul app on the computer, and Nexul reaches T3 Code through it. The computer needs Linux or macOS with an account that can use `sudo`, Windows, or a Linux host with no systemd such as a container (see [A computer with no systemd](#a-computer-with-no-systemd)), and a way out to the internet. On a Mac, T3 Code answers only while you are logged in at it, and on Windows the Nexul app itself runs only while you are logged on (see [What the command does on the computer](#what-the-command-does-on-the-computer)).

1. Open your settings, **Computers**, and press **Add a computer**. The [Welcome wizard](/docs/guide/welcome-wizard/)'s **Your computer** step runs the same steps.
2. Copy the command under **macOS / Linux** and run it in a terminal on that computer, from your own account (not as root):

   ```sh
   curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- <token>
   ```

   On Windows, copy the command under **Windows** and run it in PowerShell from your own account; it needs no administrator:

   ```powershell
   & ([scriptblock]::Create((irm https://nexul.io/computer.ps1))) <token>
   ```

   In a container, or on a Linux host with no systemd, copy the second command, **Container or no systemd**, instead. It is the same command without `sudo` and with `--foreground` at the end, and it stays running in that terminal (see [A computer with no systemd](#a-computer-with-no-systemd)). The first command picks that mode on its own when it finds systemd isn't running, but the second says so up front.

   The token in it is for this computer only and works once, within the hour the dialog names. `sudo` asks for your password once, to install a background service; everything else is installed for you, the account that typed `sudo`, and runs as you, never as root.
3. Watch the three checks. **Computer connected** passes once the Nexul app on the computer connects, **T3 Code found** once it finds T3 Code answering there, and **Paired** once Nexul has paired with T3 Code through the app's own connection. There's no pairing link to paste. A check that can't pass says what to do in its line: **Reconnecting**, **Offline**, **Closed** (open the T3 Code desktop app), **Not running**, **Missing** (run the command again without `--no-t3`), or **Failed** with the reason.
4. Press **Set up**, then [set up the computer](/docs/guide/computer-setup/), and **Done**.

If you won't run the command after all, the code stays good for an hour. To cancel it, find it under **Unused computer codes** in the **Computers** list and press **Revoke**; only you see it there. **Add this computer again** then makes a fresh command.

The computer is named after its hostname; **Rename** in its row's **…** menu changes that. If you close the dialog before running the command, the row reads **New computer** with **Add this computer again**, which makes a fresh command for the same row.

### What the command does on the computer

`sudo` is there only to place a system service. Everything else is installed for you, the person who typed `sudo`:

- the `nexul` command in your `~/.local/bin`, and the Nexul app with its credential in `~/.local/share/nexul` (`~/Library/Application Support/nexul` on a Mac), all owned by you, with the credential readable only by you;
- `nexul-computer`, a system service that runs the app as you, never as root: a systemd service on Linux, a LaunchDaemon in `/Library/LaunchDaemons` on a Mac. It starts at boot and keeps running after you log out.
- T3 Code, when it's missing; see [What the command installs](#what-the-command-installs).

Logged in as root, the command refuses: run it from your own account. On a computer without `sudo`, the shell says `sudo` is not found; as root, install sudo and add your account to the `sudo` group (`usermod -aG sudo <you>` on Debian and Ubuntu, the `wheel` group elsewhere), log in again, and rerun the command. Its logs are in `journalctl -u nexul-computer`, or `~/Library/Logs/nexul/nexul-computer.log` on a Mac. To remove it, remove the computer in Nexul, or run `nexul uninstall computer` from the same account: it tells Nexul, and a small root service the install left (a root LaunchDaemon on a Mac), which can only remove this app, takes the service and its files away; no sudo rule is involved. With sudo it removes them at once (`sudo ~/.local/bin/nexul uninstall computer`, since sudo does not search `~/.local/bin`). An agent can add a computer too, with [`computer_create`](/docs/guide/mcp-server/), which returns the same command.

On Windows the command needs no administrator, because everything it installs is yours:

- the `nexul` command and the Nexul app with its credential in `%LOCALAPPDATA%\Nexul`, with that folder added to your own Path;
- `nexul-computer`, a scheduled task of your own that starts the app when you log on and runs it as you, without administrator rights, until you log off. Windows runs a service under your account only with your password, which the command never asks for, and T3 Code's desktop app runs only while you are logged on anyway. The task starts the app again 5 seconds after it fails.
- T3 Code's desktop app through `winget` when no T3 Code is found; open it once, and Nexul pairs it.

Its log is `%LOCALAPPDATA%\Nexul\computer\service.log`. To remove it, remove the computer in Nexul, or run `nexul uninstall computer`: it tells Nexul, ends and deletes the task, and deletes `%LOCALAPPDATA%\Nexul` and its Path entry once the command ends.

### A computer with no systemd

A container, or any Linux host that isn't running systemd, has no service manager to register the Nexul app or T3 Code with. There the command registers nothing: it detects that systemd isn't running (`computer.sh` does too, and passes `--foreground` on), or you force it with `--foreground`, and it needs no `sudo`. The dialog's **Container or no systemd** command is this one. Run it as the account the app is for (a root login in a container is that account), from a shell you can leave open:

```sh
curl -fsSL https://nexul.io/computer.sh | sh -s -- <token> --foreground
```

It lays out the same files in that account's home as above, and installs T3 Code's command line when it's missing, with T3 Code's own installer. T3 Code needs `libatomic1`, which only root can add, so install it first (`apt-get install libatomic1` on Debian and Ubuntu). It then stays running in that terminal as `nexul run computer`, Nexul's own supervisor, which:

- runs the Nexul app, and `t3 serve` on port 3773 bound to `127.0.0.1` when the command installed or found a `t3` command, with the same settings a service would give them. A T3 Code that already answers is left to you, and the supervisor starts no second one. `--no-t3` leaves T3 Code out, as everywhere else;
- starts either one again 5 seconds after it exits, so a crash costs a few seconds of **Offline**. The app leaves T3 Code's restarts to it and never runs `t3 service restart`;
- passes Ctrl-C and `SIGTERM` to both and exits once they have stopped, and logs what it does to the terminal as JSON lines; the Nexul app keeps its own `runner.log` as before.

Nothing starts it after a reboot: the command prints, once, the line that does (`~/.local/bin/nexul run computer`). Put that in a container's entrypoint, or in an `@reboot` crontab entry, for the account. The app still updates itself in place, and the supervisor keeps going through it; it also takes the new binary the next time it starts the app. Only one supervisor runs per account: a second `nexul run computer` says which pid holds the first.

Removing the computer in Nexul makes the app ask its supervisor to clean up: once the app exits, the supervisor stops T3 Code, deletes the folder and the `nexul` command, and exits. `nexul uninstall computer` from the same account does the same on demand, stopping a supervisor that's running first. If the supervisor is killed with `SIGKILL`, the app and T3 Code it started keep running, so stop them before starting it again. `nexul status` doesn't list a computer, with or without systemd.

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

## When the network only allows port 443

Some networks, behind a company proxy for example, let a computer reach the internet only on port 443, the port of ordinary HTTPS. Two different things then behave differently:

- **Nexul works.** The Nexul app on the computer opens its one connection out to your Nexul address over HTTPS, so it needs nothing but port 443 to that address. Everything Nexul does with the computer goes over that connection: it reaches T3 Code through it, and nothing has to reach into the computer. You can chat with the agent from Nexul's web and phone apps, run [plays](/docs/guide/plays/), run [setup](/docs/guide/computer-setup/), let the agent use [Nexul's MCP server](/docs/guide/mcp-server/), pick T3 projects, clone a repository onto the computer, and read the computer's facts, containers and logs.
- **T3 Connect may not.** T3 Connect is T3 Code's own way to open T3 Code from another device, and it is not part of Nexul. Its tunnel connects out to Cloudflare on port 7844, with TCP and UDP, so a network that allows only 443 blocks it. T3 Code's own [troubleshooting page](https://github.com/pingdotgg/t3code/blob/main/docs/user/t3-connect-troubleshooting.md#port-7844-is-blocked-on-the-host) says so and suggests allowing port 7844 or using a direct route such as Tailscale instead.

What Nexul does **not** give you is T3 Code's own screen on that computer. There is no way to open T3 Code's web page, its desktop app or its phone app on the computer through Nexul: Nexul uses its connection to T3 Code for the agent work above and does not pass T3 Code's own interface on to your browser. If you need T3 Code's own interface while you are away from the computer, use one of T3 Code's own routes. If you only need to talk to the agent, Nexul's chat does it, and the agent's work shows in the same thread in T3 Code when you are back at the computer.

The Nexul app only needs to connect to your Nexul address. When the network makes it go through a named proxy, it uses the standard `HTTPS_PROXY` variable from its environment; the install does not set it for you. A network that lets port 443 out directly needs nothing.

## Keep it paired

Each computer is one row. A row shows a button only when something needs doing; everything else is in its **…** menu. Click the name to unfold the details.

A computer added with **Add a computer** reads each part on its own: **Online**, **Reconnecting** for up to a minute after its connection drops (a computer that was asleep or paused for a moment comes back on its own, and its runs carry on), or **Offline** (with when it was last seen) for the computer itself, then **T3 Code answering**, **Open T3 Code** when the desktop app is closed, **T3 Code not running**, **Pairing failed** with the reason, or **T3 Code refused the session** when Nexul's session was revoked inside T3 Code (its connected clients list, or `t3 auth session revoke`). Nexul pairs it again on its own before the 30-day session ends, whenever the app is connected. Its buttons are **Add this computer again** when the app is gone from it, **Re-pair now** when a pairing failed, ran out or was refused, and **Set up** or **Update skills** for its setup; the **…** menu holds **Set up** or **Re-run setup**, **Re-pair now**, **Rename** and **Remove**. Its details show T3 Code's state (**Running** with its port and version, **Not running**, or **Not installed**), then its facts.

**Add this computer again** shows when the computer has no Nexul app: the command was never run, or the app was removed because its owner's account was disabled. It opens the same dialog with a fresh command for the same row, so the computer keeps its name, project links and setup, and Nexul pairs it again once the app connects.

### Computers paired before the Nexul app

A computer paired through a Cloudflare tunnel or by URL before **Add a computer** existed keeps its row, name, project links, defaults, setup and MCP token, but Nexul no longer reaches it there: pairing that way was retired, so no chat turn, play or setup runs on it until it moves. Its row reads **Move this computer to the Nexul app**, its details still show its address and T3 Code version, and **Remove** is its only other action. Nexul has already deleted its Cloudflare tunnel, DNS record and Access app. New computers are only added with **Add a computer**.

### Move a computer to the Nexul app

1. On the computer's row, press **Move**. The dialog makes the same one-time command as **Add a computer**, made for this computer.
2. Run it on that computer from your own account, as in [Add a computer](#add-a-computer). It installs the Nexul app, and T3 Code only when it's missing.
3. Watch the three checks. **Paired** passes once Nexul has paired T3 Code through the app.
4. Once it's paired, the dialog shows the command that removes `cloudflared` from a computer that was paired through a tunnel, which needs your `sudo`:

   ```sh
   sudo cloudflared service uninstall
   ```

   On Windows, run `cloudflared service uninstall` in an administrator PowerShell. A computer paired by URL has nothing to remove.

The computer keeps its row, name, project links, defaults, setup, MCP token and trails; from then on it reads like any computer added with **Add a computer**. If pairing through the app fails, the computer stays as it was: the row says **Not moved yet** with the reason, and **Show progress** opens the checks with **Pair now** to try again. Nexul also tries again whenever the app reports, when it connects and every 6 hours.

An agent can move a computer too: [`computer_create`](/docs/guide/mcp-server/) with the computer's `id` returns the command.

### The computer's facts

A computer's Nexul app reports on it when it connects and every 6 hours: its hostname, OS and architecture, the app's version, T3 Code's state, port, version and how it is installed, `cloudflared`'s version when it is there, whether Docker answers you there and its version, your git name and email, whether the computer can sign in to GitHub, and the free disk space in your home folder. Nexul adds what T3 Code lists through the app: each provider's CLI version, whether it is signed in, its models, and T3 Code's projects with their folders. They are stored only when something changed, and **Facts changed** says when that was.

**GitHub** reads **Signed in** when the GitHub CLI is signed in, **Credential helper** when git has one configured, and **Not signed in** when the computer has neither. Nexul gives a play no GitHub write access of its own: a play that pushes a branch or opens a pull request does it with the computer's own sign-in, so run `gh auth login` on the computer (or set up a git credential helper) before you rely on one. A computer whose app is older than this report shows no **GitHub** line.

Facts are yours alone. Nobody else sees them, a workspace Owner included, and an agent reads them through `computer_list` only when it acts for you.

### Containers on the computer

When Docker answers you on the computer (Docker Desktop, Colima, or your account in the `docker` group), the computer's details have a **Containers** section listing every container there with its image and state. **Logs** opens the stack page's log viewer on that container: the last 200 lines, then new ones as they come, with the same pause, errors filter, copy and download. Nexul reads the output live from Docker through the Nexul app and stores none of it; closing the viewer stops the read on the computer. The limits are a deploy runner's: at most 16 open log views per computer, 1000 lines of history, and 64 KB a second, past which lines are skipped and counted.

These are your own containers, not Nexul services: nothing is imported, deployed, or drawn on Topology. Only you see them, unmasked; anyone else, a workspace Owner included, gets not found, unless you [share the computer with Read logs](#share-a-computer), which shows them its containers with each container's own environment values hidden. Docker is checked when the app connects and every 6 hours, so a Docker started later shows after the app's next report, or at once when the app restarts, as it does when the computer does. A computer without Docker has no Containers section. An agent acting for you reads the same through [`computer_list`](/docs/guide/mcp-server/) with the computer's `id`.

### The Nexul app's own log

On a computer you add with **Add a computer**, the Nexul app also writes its own log to `runner.log` in its own folder next to its credential (`~/.local/share/nexul/computer` on Linux, `~/Library/Application Support/nexul/computer` on a Mac, `%LOCALAPPDATA%\Nexul\computer` on Windows), readable only by you. At 5 MB it starts a new file and keeps the one before it, so there are never more than two. The service manager's own output is unchanged: `journalctl -u nexul-computer` on Linux, the log file on a Mac, `service.log` on Windows.

Nexul reads it live through the app and stores none of it. An agent acting for you reads the last lines through [`computer_list`](/docs/guide/mcp-server/) with the computer's `id` and `runner_logs: {lines}`, 200 by default and at most 1000; the HTTP gateway serves the same at `/api/pairing/computers/<id>/runner/logs`, with a live tail on a WebSocket beside it. Each record is one line, `time level message key=value…`, with any Nexul token hidden. The limits are the containers' above. Only you read it, unless you [share the computer with Read logs](#share-a-computer).

**Remove**, in the **…** menu, asks first, then deletes the pairing and revokes the computer's MCP token. On a computer added with **Add a computer**, it also revokes the Nexul app's credential: the app removes its `nexul-computer` service and files within a heartbeat, or the next time the computer comes online, and on the way out ends every T3 Code session labelled `Nexul`, so Nexul keeps no way into T3 Code there. A command made for the computer and never run stops working too. On a computer paired before the Nexul app, Remove deletes its row; Nexul's session inside T3 Code there stays until it expires, or until you run `t3 auth session list` on the computer and `t3 auth session revoke <id>` on the `Nexul` entry. Re-pair never ends a session.

### The computer's MCP token

Setup gives the computer its own personal access token, "Nexul MCP on <computer>", for its providers to reach Nexul. It shows in the row's details. **Replace** mints a new one, shown once, and **Revoke MCP token** cuts it off. With none, press **Mint MCP token**.

## Share a computer

A computer is yours alone until you share it: nobody else sees it, a workspace Owner and the instance's Owner included. To let someone use it, open **Share…** in its row's **…** menu, press **Add a person**, pick them, and set what they may do, then **Save**:

- **See this computer**: it shows in their **Computers** list under **Shared with you**, as "<you>'s <computer> (shared)", with its name and whether it is online. Its facts, setup, containers, MCP token and everything else stay yours.
- **Run agents** and **Run commands**: allowing either also lets them see it. A shared computer shows in their run dialog, where it can't be picked yet, and stays out of their order of computers, so it takes none of their runs.
- **Read logs**: this runner's logs and its Docker containers' logs. It is off unless you allow it, and allowing it also lets them see the computer. They then read the Nexul app's own log, list the computer's containers and read each one's logs; the values of a container's own environment that are six characters or longer show as `••••` in what they read, which your own reads never do. A container's output can still show a secret the app derives or fetches by itself, so allow it only for people you would trust with that.

Each permission is **Allow** or **Fallback**; there is no Deny on a computer. **Fallback** on **See this computer** takes the other three away too, and the bin beside a person in the list, pressed twice, takes them off. **Check access** shows what each person can do on it.

Read the warning at the top of the dialog before you share: agents and commands on your computer run as your own user there, so whoever you let run them can read what your account can, your SSH keys, git credentials and logins included.

- You can share only with people who share a workspace with you, and only with people, never a role.
- Only you change who it's shared with. The people you share it with can't share it on or raise their own permissions, and a change takes effect at once.
- Sharing ends for good when the two of you no longer share a workspace, or either account is disabled or removed: adding them back, or turning the account back on, doesn't bring it back, so share it again. Removing the computer ends all of its sharing.

An agent shares a computer with [`permission_overwrite_update`](/docs/guide/mcp-server/), `resource_type` `computer`, the people's `user_ids`, and `computers:see`, `computers:run_agents`, `computers:run_commands` or `computers:read_logs`; only the computer's owner can. `computer_list` lists the computers shared with you after your own.

## Choose where turns run

The **Computers** settings page has two tabs, **Computers** and **Projects**.

### Your computers, in order

**Your computers** on the Computers tab is one ordered list, and runs go to the first computer in it that can take one: it is online, set up for the run's provider, holds the project, and has room. Drag a computer by its grip to change the order, or use **Move up** and **Move down** in its **…** menu. A computer that can't take the run is passed over for the next one, so a laptop that is shut sends its runs to the computer after it. Computers shared with you are not in the list; they run nothing of yours unless an auto play [names one](/docs/guide/plays/#run-on-a-chosen-computer).

Above the list, one line says where `@Agent` chats outside a project start, or why they can't, such as "Chats outside a project can't start: laptop is offline; build-box has no chat folder." It is only about those chats: each computer's own row says whether it can take a project's runs.

Each computer's details set two things for it:

- **Runs at once**, 1 to 5: how many play runs it takes side by side. Its row says how many are going, such as "1 of 2 running". A full computer is passed over like an offline one. Only play runs count; chats never fill a computer.
- **Chat folder**: the T3 project `@Agent` works in from a channel or direct message, where there is no project to find.

### Defaults and each project

The Projects tab is one list. **Defaults** comes first and applies wherever a project says nothing of its own; every project you can open follows, from all your workspaces, each reading back what your runs there use, or **Uses defaults**. Open a row to change it; one row is open at a time.

Defaults set:

- **Send runs to**: **First in order** keeps your first computer the main one and uses the rest only when it can't; **Least busy** sends each run to the computer with the smallest share of its runs at once in use, in order on a tie.
- **Model**, with its options. **Computer default** leaves it to the computer. A computer that doesn't offer the model is passed over.
- **Threads start in**: **Project folder**, or **New worktree per thread** so runs side by side never edit the same files.

The model list comes from the first of your computers that is online. When none is, the model shows what is saved and reads "Turn on a computer to change it."

A project's row shows, for each computer in order, the checkout a run there would use. Nexul finds it by matching each of the computer's T3 projects' git remote to a repository of the project, so a folder can have any name. When a computer holds several checkouts of the project, **Pick a checkout** says which one, and the pick is remembered. A computer that has none reads **No checkout**, and **Add** beside it clones the project's repository there, or creates an empty T3 project when there is nothing to clone. The clone lands beside the computer's other T3 projects, or in `~/Code` when it has none. The Nexul app on the computer clones it with a short-lived, read-only token for that one repository, so the computer needs no GitHub sign-in to clone a private one; pushing from it later still does (`gh auth login`, for one). Without the GitHub App's private key, or with a Nexul app older than this, T3 Code clones with the computer's own git sign-in instead. A computer paired before cloning existed shows "Pair it again to clone", with a **Pair again** button that fixes it in one click.

**Computers** is **Your order** until you pick **Custom**, which gives the project its own list: only the computers in it, in its own order, with its own **Send runs to**. Use the **…** menu to move or remove one, and **Add a computer** to put another in. The project can also pick its own model and where its threads start; anything left on **Same as defaults** follows them. **Save** keeps the changes; **Use defaults** clears them. What you set here is yours alone: each teammate's runs in the same project follow their own computers.

### When it is chosen

A computer is chosen when a run starts, never moved after: a run that has started stays on its computer through its answers, **Continue this run**, and a Nexul restart, even if you change the order meanwhile. A run that waits in the queue is placed when its turn comes, so it goes to whichever computer can take it then. In a chat, `@Agent` on a ticket's or doc's thread works in that project's checkout; in a channel or direct message, and in a project none of your computers holds, it works in the first ready computer's **Chat folder**. A thread that already exists keeps its computer.

The trail of a run that passed computers over lists them under **Passed over**, with why, such as "laptop is offline". Only you see it, on your own runs.

When Nexul was upgraded to this order, your old default computer went first and the rest followed newest first; each computer's runs at once took the workspace's old setting, and your old fallback T3 project became that computer's chat folder. Your project links kept their model and where threads start, and now follow your order instead of one computer.

## When a turn can't start

The page says why once, above or beside its play buttons, and each button repeats it as its tooltip. In chat, the message says why:

| Message | Fix |
|---|---|
| No computer is paired to run plays. | Add a computer, or finish adding one. |
| Your computer's pairing has expired or was revoked. | Press **Re-pair now** on the computer's row. |
| None of your computers has this project yet. | Pick a computer and its T3 project in the run dialog, or **Add** it to a computer on the Projects tab. |
| None of your computers can take a run right now. | Fix one of the reasons it lists for each computer, below. |
| @Agent can't use a provider on a computer until its setup is done. | Run setup for that computer. |
| T3 Code on your computer is offline. | Check the computer is **Online** and T3 Code is running on it. |

A run that none of your computers can take says why for each one:

| Reason | Fix |
|---|---|
| laptop is offline | Start the computer, or check the Nexul app is running on it. |
| T3 Code isn't answering on laptop | Check T3 Code is running on it. |
| laptop is full (2 of 2 running) | Wait for a run to end, or raise its **Runs at once**. |
| laptop isn't set up for claude | Run setup for that provider on it. |
| laptop doesn't offer the model | Pick a model the computer has, or leave the model empty. |
| laptop doesn't have this project | **Add** it on the Projects tab. |
| laptop has several checkouts of this project; pick one | **Pick a checkout** on the Projects tab. |
| laptop has no chat folder | Set its **Chat folder**. |
| laptop needs pairing again | Press **Re-pair now** on its row. |
