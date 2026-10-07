---
title: Upgrade
description: Move the instance to a new release from the web app or the server, roll back, and restore a database snapshot.
sidebar:
  order: 2
---

An upgrade swaps the binary of every Nexul service on the machine and restarts each one. Expect under a minute of downtime.

## From the web app

1. Open **Settings → Instance**. **Instance version** shows what you run, its channel, and whether a newer release exists.
2. Click **Upgrade to vX** and confirm.

The `instance` runner runs the upgrade on the server, so it keeps going while the server restarts. The page reconnects by itself and shows **Upgraded to vX**. The `instance_upgrade` MCP tool does the same.

The button is missing when the instance is a development build, is already on the newest release, or its `instance` runner is offline or busy. The section says which.

Each download is retried a few times, so a release host's brief outage doesn't stop an upgrade. If the upgrade still stops before the restart, the section reports it as failed within seconds, shows the error, and offers the upgrade again. If the instance is still on the old version after fifteen minutes, the section reports the upgrade as failed too. Read the upgrade's output on the server:

```sh
journalctl -u 'nexul-upgrade-*'
```

On a Mac it is in `~/Library/Application Support/nexul/nexul-upgrade.log`, on Windows in `%ProgramData%\Nexul\nexul-upgrade.log`.

## From the server

```sh
sudo nexul upgrade
```

This takes the newest release on your channel (stable, or beta if you installed a beta), checks it against the release's `checksums.txt`, and moves every Nexul service on this machine to it. A service already on that release keeps running. `nexul status` shows each service's version afterwards.

Runners on other machines update themselves the next time they connect to the upgraded server (see [Runners](/docs/guide/runners/#updates)). An automations host on another machine moves when you run `nexul upgrade` on that machine.

## Pinning and rolling back

```sh
sudo nexul upgrade --version v0.2.0
```

This installs that exact release, newer or older than what runs now. If the rollback crosses a database change, [restore a snapshot](#restoring-a-database-snapshot) too.

## What happens to the database

Before the server applies a database migration, it copies the database to `data/backups/` in the install directory and keeps the newest five copies. No migration, no copy.

An older server refuses to start on a database a newer one has migrated. Its error names the snapshot to restore. There are no downgrade migrations: the snapshot is the way back.

## Restoring a database snapshot

With the default install directory, `/data/nexul`:

1. Stop the server:

   ```sh
   sudo systemctl stop nexul-server
   ```

2. Copy the snapshot over the database and delete the WAL files, so SQLite doesn't replay them onto the restored copy:

   ```sh
   sudo sh -c 'cd /data/nexul/data && cp backups/<snapshot-file> nexul.db && rm -f nexul.db-wal nexul.db-shm'
   ```

3. Install the release you ran before the upgrade, which starts the server again:

   ```sh
   sudo nexul upgrade --version <previous-version>
   ```

Snapshot files are named `nexul-<version>-<time>.db`. The version is the release that took the snapshot just before migrating, so the file holds the database as your previous release left it.
