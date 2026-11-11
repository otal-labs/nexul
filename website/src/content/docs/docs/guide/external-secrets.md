---
title: External secrets
description: Keep secrets in 1Password and let Nexul hold only a pointer to each one, read by the runner, automations host or server that uses it.
sidebar:
  order: 6
---

Instead of pasting a secret into Nexul, you can paste a **secret reference**: the address of the secret in 1Password. Nexul stores the address. The side that needs the secret reads it from 1Password at that moment, with a credential that lives only on its own machine. A copy of Nexul's database, or a breach of the server, gives nobody a way into your vaults.

1Password is the only manager built so far.

## Writing a reference

A value is a reference when the whole value is

```
op://vault/item/field
op://vault/item/section/field
```

for example `op://Acme Prod/Database/password`. Anything else is a plain value, including `postgres://…` and every other URL. A reference is never read inside a longer value, so `postgres://app:op://Acme Prod/Database/password@db` is a plain value. To build a connection string, store the whole string in 1Password, or compose it with `${VAR}` in the stack's compose file.

Paste a reference wherever you would paste a secret:

- a stack's environment, and the overrides of a branch deploy rule;
- the automation secrets pool;
- the secret fields of a connector, and a sign-in provider's client secret.

The field shows a **1Password** tag when it holds a reference. A reference that is shaped wrong (no vault, no item or no field) is refused when you save. Whether the item exists is only known when it is read.

A reference is kept as closely as a secret: people without `stacks:write` see only the variable's name, the automation pool never shows it again after saving, and events, agent results and the audit log never carry it.

The one value that can't be a reference is `NEXUL_AUTH_SECRET`. It signs sessions and seals every stored secret, so the server can't wait on 1Password to start.

An existing value that already starts with `op://` becomes a reference when you upgrade to the release that adds this.

## Make a service account for each reader

A **service account** is 1Password's credential for software. Make one for each of:

- every runner that deploys stacks with references, the one `nexul install` puts on your server (named `instance`) included;
- every automations host whose workspace uses references in its pool;
- the server, for connectors and sign-in providers.

Give each one read access to only the vaults it needs. Someone with `stacks:write` can aim a reference at anything the runner's account can read and deploy a container that prints it, so a runner on a staging machine should read a staging vault and nothing else. For the server, use a vault that holds only Nexul's own credentials.

The token goes only on the machine that uses it. Nexul never holds a runner's or a host's token and never sees a value they read.

## Set a token

Every command reads the token from standard input, or asks for it without showing it when you type it. A token is never taken as a command-line value. Each command also rotates a token, and `--remove` takes it away. Each restarts the service, and `nexul upgrade` keeps the token.

**A runner**, when you add it (see [Runners](/docs/guide/runners/#a-1password-token)) or later:

```sh
sudo nexul install runner --server https://nexul.example.com --name build-box-1 --code nxe_… --1password-token-file ./token
printf '%s' "$OP_TOKEN" | sudo nexul runner 1password build-box-1
sudo nexul runner 1password build-box-1 --remove
```

The runner `nexul install` put on the server is named `instance`. A computer someone added for their own agent work never deploys, so it takes no token.

**An automations host**, for the one workspace it serves:

```sh
printf '%s' "$OP_TOKEN" | sudo nexul automations 1password builder --workspace acme
sudo nexul automations 1password builder --remove
```

**The server:**

```sh
sudo nexul install --1password-token-file ./token
printf '%s' "$OP_TOKEN" | sudo nexul server 1password
sudo nexul server 1password --remove
```

An interactive `nexul install` also asks for a server token and a token for the runner it installs: paste one, or press Enter to skip. Without a terminal it never asks, and nothing is needed at install.

### One workspace per host that holds a token

Every automation on a host runs in the host's own process, as the same operating system user, so any of them can read the host's token. A host that holds a token for a workspace therefore runs only that workspace's automations, and they can only reach that workspace's vaults.

- `nexul automations 1password` asks the instance which workspaces the host serves. It refuses a token for a second workspace, and a token for a host that has another workspace's automations placed on it; move those to another host first.
- Nexul refuses to create, move or switch on an automation on a host holding another workspace's token, and says `<host> holds a 1Password token for another workspace`.
- The `instance` host runs every workspace's default automations, so it can hold a token only while the instance has one workspace. A workspace made after that gets its defaults switched off.
- The token is kept against the workspace's id, so renaming the workspace doesn't hand it to another one.

## When it is read

- **A deploy.** The runner reads every reference of the deploy in one batch, before it pulls, builds or starts anything, every time. If any fails, the deploy fails with the running containers untouched, naming each variable and why (`DB_PASSWORD: field not found`), never a reference or a value. **Retry** tries again. A deploy with references goes only to a runner on the stack's machine that reports **1Password** as ready; if none ever has, the deploy is refused with `no runner on <machine> can read 1Password references`. Rollback and branch deploys use each reference's current value.
- **An automation.** The host reads the workspace's pool when the automation connects. If a reference fails, the automation still starts with every secret that did resolve; the failed name is missing from `ctx.secrets`, and the automation's log and page say `NAME: field not found`.
- **The server.** It reads connector and sign-in references when it starts and when you save them, then keeps them in memory. If a provider answers 401 or 403, it reads that credential again once (at most every 5 minutes) and retries. A restart reads everything again.

If 1Password can't be reached when the server starts, it still starts. The connector or sign-in button shows **1Password unreachable** and retries with backoff, and passkeys and signed-in sessions keep working.

## Noticing a changed secret

Someone changes a password in 1Password, and the running containers still have the old one. A runner and an automations host check for this every `NEXUL_SECRET_CHECK_INTERVAL`: `1h` unless you set another, never under `5m`, or `off`. Set it in the service's environment and restart the service. A value under five minutes stops the service from starting, because each check is a read against your 1Password limits.

- **A stack:** the runner compares what it reads now with what the last deploy used. The stack's page, on the web and the phone, shows **DB_PASSWORD changed since the last deploy** beside **Redeploy**. Only variable names are sent, never a value or a reference. Nothing redeploys by itself; the next deploy, retry or rollback clears the notice. Automations can react to the `service.secrets_changed` event.
- **An automation:** the host restarts it as soon as no run is in flight, or after 5 minutes at most. A run cut off by the restart is delivered again.

The server has no timed check.

## What the states mean

A runner's row on **Runners**, an automations host's row, a connector's card and **Settings → Connectors** show a tag for each manager:

| Shown | Means |
| --- | --- |
| **1Password** | The token works. |
| **1Password token rejected** | 1Password refused the token: it was deleted, expired or mistyped. Everyone who can **Manage** the machine gets one inbox notice naming the command to set a new token, and the instance Owner gets it for the server. |
| **1Password unreachable** | 1Password couldn't be reached to try the token. It is tried again. |
| No tag | No token is set. |

A stack with references on a machine whose runners can't read them shows **No runner on prod can read 1Password references yet**, and its deploys are refused. An automation whose host can't read its pool shows **Host can't resolve secret references**.

## What a failure says

| Reason | Means |
| --- | --- |
| `field not found` | The vault and item exist, but the item has no such field. Check the field's name, and the section if the reference has one. |
| `not found` | No such item. |
| `no access` | The service account can't read it. |
| `1Password unreachable` | 1Password couldn't be reached, so nothing was read. Try again. |
| `1Password token rejected` | The token is no longer valid. Set a new one. |

A vault the service account can't reach answers exactly like a vault that doesn't exist, so a token never learns which vaults it lacks. If a reference is right, check that the service account has read access to its vault.

## Read limits

1Password limits how often a service account reads. On the plans measured, each token may read 1,000 times an hour, and the whole 1Password account 1,000 reads and writes a day, shared by every token. Check your own with `op service-account ratelimit`.

A batch is one read whatever its size, so a deploy costs one read however many references it holds, and a host restart costs one per workspace. The check at its default costs about 24 reads a day for each runner and each host token. A large fleet on a small plan should raise the interval, or switch the check off.

## See also

[Stacks and deploys](/docs/guide/stacks-and-deploys/#environment), [Runners](/docs/guide/runners/#a-1password-token), [Automations](/docs/guide/automations/#secrets) and [GitHub App and connectors](/docs/guide/github-app/).
