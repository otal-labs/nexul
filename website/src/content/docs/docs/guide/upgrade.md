---
title: Upgrade
description: How to upgrade a running instance, pin or roll back a version, and restore a database snapshot.
sidebar:
  order: 2
---

Nexul ships prebuilt binaries, so an upgrade replaces the `nexul` command with the release's, swaps the binary of every Nexul service on the machine, and restarts each one. You can start it from the web UI or from the server.

## From the web UI

Open **Settings → Instance**. The **Instance version** section shows the running version, its channel, and the newest release on that channel. When a newer release exists, **Upgrade to vX** starts the upgrade after a confirmation. The same action exists as the `instance_upgrade` MCP tool.

What happens: the server asks the `instance` runner to run `nexul upgrade --detach --version vX` on the server. `--detach` starts the upgrade outside the runner's own service, so it carries on while the runner and the server restart underneath it: on Linux as a transient systemd unit named `nexul-upgrade-<suffix>`, on a Mac or Windows PC as a background process. The browser reconnects on its own and the section reports **Upgraded to vX**. Expect under a minute of downtime while the services restart.

If the instance is still on the old version fifteen minutes later, the section reports the upgrade as failed. The upgrade's output is in the journal on Linux:

```sh
journalctl -u 'nexul-upgrade-*'
```

On a Mac it is in `~/Library/Application Support/nexul/nexul-upgrade.log`, and on Windows in `%ProgramData%\Nexul\nexul-upgrade.log`.

The UI upgrade needs an install made with `nexul install`, which is what gives the server an `instance` runner and the `nexul` command on the host.

## From the server

```sh
nexul upgrade
```

This finds the newest release on your install's channel (stable, or beta if you installed a beta), downloads that release's `nexul` and checks it against the release's `checksums.txt`, then hands over to it. The new version downloads each service's binary once, copies it to every service of that kind on the machine, and restarts them one at a time: the server, OpenObserve when its pinned version changed, then each runner and automations host. A service already on the target version is left running. `nexul status` shows each service's version afterwards.

Runners on other machines need nothing run by hand: a runner updates itself the next time it connects to the upgraded server (see [Runners](/docs/guide/runners/#updates)). An automations host on another machine moves when you run `nexul upgrade` there, which also moves every runner on that machine straight away.

## Pinning and rolling back

```sh
nexul upgrade --version v0.2.0    # this release exactly, newer or older than the running one
```

Rolling back is the same command with an older version. See [restoring a database snapshot](#restoring-a-database-snapshot) if the rollback crosses a schema change.

## What happens to the database on upgrade

Before the server applies any pending migration, it snapshots the SQLite database with `VACUUM INTO` to a file in `data/backups/` in the install directory, and keeps the five newest snapshots. If nothing is pending, no snapshot is written.

An older binary refuses to start against a database with a newer schema. It names the snapshot to restore in its error rather than starting against data it doesn't understand. That refusal, plus the snapshot, is the rollback safety net: there's no separate downgrade migration path.

## Restoring a database snapshot

With the default install directory, `/data/nexul`:

1. Stop the server:

   ```sh
   systemctl stop nexul-server
   ```

2. Copy the snapshot over the live database and remove the WAL files, so SQLite doesn't replay them against the restored file:

   ```sh
   cd /data/nexul/data
   cp backups/<snapshot-file> nexul.db && rm -f nexul.db-wal nexul.db-shm
   ```

3. Start the version the snapshot came from:

   ```sh
   nexul upgrade --version <previous-version>
   ```
