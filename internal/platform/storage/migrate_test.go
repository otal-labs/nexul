package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrate_ClosedDB_Error(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	err = Migrate(db)
	require.Error(t, err)
}

func TestApplyMigration_InvalidSQL_Error(t *testing.T) {
	db := newTestDB(t)
	err := applyMigration(db, "bad", "THIS IS NOT SQL;")
	require.Error(t, err)
}

func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestPending_FreshDB_ListsEveryEmbeddedVersion(t *testing.T) {
	db := freshDB(t)

	pending, err := Pending(db)
	require.NoError(t, err)

	want, err := embeddedVersions()
	require.NoError(t, err)
	assert.Equal(t, want, pending)
}

func TestPending_FullyMigratedDB_IsEmpty(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, Migrate(db))

	pending, err := Pending(db)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestMigrate_Twice_AppliesPrivateInvitationMigrationOnceAndPreservesActiveStatus(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, Migrate(db))
	_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, provider, provider_user_id, login, created_at, updated_at) VALUES ('u-1', 'github', 'provider-1', 'alice', 1, 1)`)
	require.NoError(t, err)
	require.NoError(t, Migrate(db))

	var applied int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM schema_migrations WHERE version = '0002_private_invitations'`).Scan(&applied))
	assert.Equal(t, 1, applied, "schema_migrations is the idempotence guard for the SQLite ADD COLUMN in 0002")
	var status string
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT account_status FROM users WHERE id = 'u-1'`).Scan(&status))
	assert.Equal(t, "active", status)
}

func TestMigrateWithBackup_WritesABackupOnlyWhenMigrationsArePending(t *testing.T) {
	db := freshDB(t)
	dir := filepath.Join(t.TempDir(), "backups")

	require.NoError(t, MigrateWithBackup(db, dir, "v0.2.0"))
	names, err := backupNames(dir)
	require.NoError(t, err)
	assert.Len(t, names, 1, "a fresh db has pending migrations, so a backup is written")

	require.NoError(t, MigrateWithBackup(db, dir, "v0.2.0"))
	names, err = backupNames(dir)
	require.NoError(t, err)
	assert.Len(t, names, 1, "nothing pending on the second run, so no new backup")
}

func TestBackup_FileOpensWithTheSameTables(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, Migrate(db))
	dir := t.TempDir()

	path, err := Backup(db, dir, "v0.2.0")
	require.NoError(t, err)
	assert.FileExists(t, path)
	assert.NoFileExists(t, path+".tmp")

	backupDB, err := OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backupDB.Close()) })

	var originalTables, backupTables []string
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	require.NoError(t, err)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		originalTables = append(originalTables, name)
	}
	require.NoError(t, rows.Close())

	rows, err = backupDB.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	require.NoError(t, err)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		backupTables = append(backupTables, name)
	}
	require.NoError(t, rows.Close())

	assert.NotEmpty(t, originalTables)
	assert.Equal(t, originalTables, backupTables)
}

func TestPruneBackups_KeepsTheNewestFive(t *testing.T) {
	dir := t.TempDir()
	for i := range 8 {
		name := filepath.Join(dir, "nexul-v0.2.0-2026091"+string(rune('0'+i))+"T000000.db")
		require.NoError(t, os.WriteFile(name, []byte("x"), 0o600))
	}

	require.NoError(t, PruneBackups(dir, 5))

	names, err := backupNames(dir)
	require.NoError(t, err)
	require.Len(t, names, 5)
	assert.Equal(t, "nexul-v0.2.0-20260917T000000.db", names[len(names)-1], "newest name survives")
}

func TestPruneBackups_MissingDir_NoError(t *testing.T) {
	require.NoError(t, PruneBackups(filepath.Join(t.TempDir(), "missing"), 5))
}

func TestMigrateWithBackup_DowngradeGuard_FiresOnAnUnknownSchemaVersion(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, Migrate(db))
	_, err := db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, "9999_from_the_future", time.Now().Unix())
	require.NoError(t, err)
	dir := t.TempDir()

	err = MigrateWithBackup(db, dir, "v0.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9999_from_the_future")
	assert.Contains(t, err.Error(), "is newer than this binary (v0.1.0)")
	assert.Contains(t, err.Error(), "restore the newest snapshot in "+dir)
}

func TestMigrateWithBackup_DowngradeGuard_NamesTheNewestBackupFile(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, Migrate(db))
	_, err := db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, "9999_from_the_future", time.Now().Unix())
	require.NoError(t, err)
	dir := t.TempDir()
	older := filepath.Join(dir, "nexul-v0.1.0-20260101T000000.db")
	newest := filepath.Join(dir, "nexul-v0.1.0-20260201T000000.db")
	require.NoError(t, os.WriteFile(older, []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(newest, []byte("x"), 0o600))

	err = MigrateWithBackup(db, dir, "v0.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), newest)
}

func TestMigrateWithBackup_ClosedDB_Error(t *testing.T) {
	db := freshDB(t)
	require.NoError(t, db.Close())

	err := MigrateWithBackup(db, t.TempDir(), "v0.2.0")
	require.Error(t, err)
}
