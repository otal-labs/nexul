---
title: Upgrade
description: How to upgrade a running instance, pin or roll back a version, and restore a database snapshot.
sidebar:
  order: 2
---

Nexul ships prebuilt images instead of building from source, so an upgrade is a pull of newer images and a restart of the stack. You can do that from the web UI or from the host.

## From the web UI

Open **Settings → Instance**. The **Instance version** section shows the running version, its channel, and the newest release on that channel. When a newer release exists, **Upgrade to vX** starts the upgrade after a confirmation. The same action exists as the `instance_upgrade` MCP tool.

What happens: the server asks the bundled `instance` runner to start a one-shot helper container named `nexul-upgrade`. The helper pulls the release's images and runs `docker compose up -d` against the same compose project, directory, and files the stack was started from, which it reads from the runner's own container labels. The stack restarts on the new images, the browser reconnects on its own, and the section reports **Upgraded to vX**. Expect about a minute of downtime while containers restart.

If the instance is still on the old version fifteen minutes later, the section reports the upgrade as failed. The helper's output is on the host:

```sh
docker logs nexul-upgrade
```

Limits of the UI upgrade:

- It updates images only. If a release changes `docker-compose.yml`, its notes say so; run the manual upgrade below for that release.
- The helper pulls without registry credentials, so the images must be publicly pullable.
- It needs the bundled `instance` runner, which only exists on compose installs. Single-binary installs replace the binary by hand.

## From the host

```sh
git pull
./install.sh        # Linux / macOS
```

```powershell
git pull
.\install.ps1        # Windows
```

`git pull` picks up any change to `docker-compose.yml` itself; the script then runs `docker compose pull` followed by `docker compose up -d`, which pulls the newest images for your channel and restarts the stack on them. Runners on other machines update themselves the next time they connect to the server — there's nothing to run on them by hand. The instance runner (the one bundled in the stack) updates as part of the same `docker compose pull`.

## Pinning and rolling back

Set `NEXUL_VERSION` before running the install script to control which images it pulls:

```sh
NEXUL_VERSION=0.2.0 ./install.sh    # pin to a specific release (image tags carry no leading v)
NEXUL_VERSION=beta ./install.sh     # track the beta channel (a new beta per merge to master)
```

Without it, `docker-compose.yml` defaults to `latest`: the newest stable release, or the newest beta while no stable release has been published yet. Rolling back is the same mechanism: pin `NEXUL_VERSION` to an older tag and run the script again. See [restoring a database snapshot](#restoring-a-database-snapshot) below if the rollback needs to cross a schema change.

## What happens to the database on upgrade

Before the server applies any pending migration, it snapshots the SQLite database with `VACUUM INTO` to a file in `backups/` next to `nexul.db`, and keeps the five newest snapshots. If nothing is pending, no snapshot is written.

An older binary refuses to start against a database with a newer schema — it names the snapshot to restore in its error rather than starting against data it doesn't understand. That refusal, plus the snapshot, is the rollback safety net: there's no separate downgrade migration path.

## Restoring a database snapshot

1. Stop the stack:

   ```sh
   docker compose down
   ```

2. Find the data volume's full name (Compose prefixes it with the project name, usually the checkout directory), then copy the snapshot over the live database and remove the WAL files so SQLite doesn't try to replay them against the restored file:

   ```sh
   docker volume ls --filter name=nexul-data
   docker run --rm -v <full-volume-name>:/data alpine \
     sh -c 'cp /data/backups/<snapshot-file> /data/nexul.db && rm -f /data/nexul.db-wal /data/nexul.db-shm'
   ```

3. Pin the version the snapshot came from:

   ```sh
   NEXUL_VERSION=<previous-version> ./install.sh
   ```

4. Start the stack:

   ```sh
   docker compose up -d
   ```
