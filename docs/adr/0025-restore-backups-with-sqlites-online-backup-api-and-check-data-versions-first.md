# 0025. Restore backups with SQLite's online backup API, and check data versions first

- **Status:** Accepted
- **Date:** 2026-10-13

## Context

Agents restore the backups they make through the CLI
([docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md]). When this
was decided, restore was CLI-only and the app had no restore. The app may be open while the CLI
restores, and the app holds an open connection to `callback.db` the whole time it runs.

Update, 2026-10-15: the app now restores any backup itself (features/restore-backups.feature),
through the same restorer, so these decisions apply to its restores too.

A backup is a whole copy of the database, made with `VACUUM INTO`. Agent backups are kept for up to
90 days, so the user may update Callback between making a backup and restoring it. The backup then
has an older schema than the live database.

Users can have more than one version of Callback on their computer. Both the app and the CLI run
goose migrations when they open the database
([docs/adr/0006-run-database-migrations-with-goose.md]), and goose records the migrations it has
applied in the database. An older version opening data that a newer one migrated would run against
a schema it doesn't know.

SQLite's online backup API copies one database into another while other connections have it open,
under SQLite's own locking. Connections that already had the destination open see the new contents
on their next read. `mattn/go-sqlite3` exposes it
([docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md]). At about a thousand jobs, the
database is roughly 12 MB, and copying it takes about a tenth of a second.

## Decisions

1. The CLI restores a backup by copying it into the live database with SQLite's online backup API,
   not by replacing the database file, and without closing or contacting the app.
2. A backup from an older schema version is copied to a temporary file, migrated to the current
   version there, and that copy is restored. The backup file itself is left unchanged.
3. A backup from a newer schema version than the CLI's is refused.
4. The app and the CLI each compare the database's highest applied migration with the highest one
   they embed before running migrations, taking a daily backup, or removing backups. If the
   database is newer, they stop without changing anything: the app tells the user to update
   Callback, and the CLI exits with the exit code for data from a newer version.

## Reasons

- **1.** The app and the CLI don't have to interface with each other. SQLite's locking makes the
  restore wait for a write the app is in the middle of, and makes the app's writes wait for the
  restore, within the busy timeout the connections already set. The app reads the restored data on
  its next query, and the board shows it when the user comes back to the window, as for any other
  change an agent makes.
- **1.** It copies the database page by page, so it doesn't depend on the schema and keeps working
  as migrations are added. It needs no new dependency.
- **2.** Restoring an older backup as is would roll the schema back under a running app, whose
  queries would then fail until it restarted. Migrating a copy keeps the backup as it was made.
- **3, 4.** An older version must never change data it doesn't understand. Reaching a backup from a
  newer version takes moving files around by hand, since the CLI would already refuse a newer live
  database. Refusing it costs only the same version comparison and protects against that.
- **4.** The check has to come before migrations, which would otherwise run against the newer
  schema, and before backups, since an older version taking or rotating daily backups would still
  change what's kept.

## Alternatives considered

- **Replace the database file with the backup.** Rejected: on Windows the file can't be replaced
  while the app has it open, and where it can, the app keeps using its old connection.
- **Have the CLI close or kill the app before restoring.** Rejected: it means the app and the CLI
  interfacing with each other directly, which adds a lot of complication.
- **Copy table by table** (`ATTACH` the backup and copy each table in a transaction). Rejected: it
  has to know every table, so it changes with each migration, where the backup API doesn't.
- **Restore an older backup as is.** Rejected: it rolls the schema back under the running app.

## Related

- [features/agent-backups.feature#restore-older-backup] A backup from an older version is brought up to date when it's restored
- [features/agent-backups.feature#restore-newer-backup] A backup from a newer version can't be restored
- [features/restore-backups.feature#app-restore-older-version] A backup from an older version is brought up to date when it's restored
- [features/restore-backups.feature#app-restore-newer-version] A backup from a newer version can't be restored
- [features/agent-backups.feature#restore-app-open-safe] Restoring while the app is open doesn't need the app closed
- [features/agent-backups.feature#restore-app-open] The app shows the restored data when I come back to it
- [features/data-version.feature#app-newer-data] The app refuses data a newer Callback has updated
- [features/data-version.feature#app-newer-data-no-backup] The app takes no daily backup of data from a newer Callback
- [features/data-version.feature#app-older-data] The app brings older data up to date
- [features/data-version.feature#app-same-version-data] The app opens data from the same version as usual
- [features/agent-cli.feature#cli-newer-data] The command line refuses data a newer Callback has updated
- [features/agent-cli.feature#cli-older-data] The command line brings older data up to date, as the app does
- [docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md] Use mattn/go-sqlite3 as the SQLite driver
- [docs/adr/0006-run-database-migrations-with-goose.md] Run database migrations with goose
- [docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md] Let users drive agent tasks from their own agent through a CLI
