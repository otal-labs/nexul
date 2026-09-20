package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// backupPrefix names every snapshot Backup writes, so PruneBackups and the downgrade guard can find them.
const backupPrefix = "nexul-"

// Backup snapshots db to <dir>/nexul-<version>-<UTC yyyymmddTHHMMSS>.db via VACUUM INTO, written to a
// ".tmp" path and renamed into place on success so a crash mid-snapshot never leaves a partial file at the final
// name. Returns the final path.
func Backup(db *sql.DB, dir, version string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	name := fmt.Sprintf("%s%s-%s.db", backupPrefix, version, time.Now().UTC().Format("20060102T150405"))
	final := filepath.Join(dir, name)
	tmp := final + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return "", fmt.Errorf("clear stale backup tmp: %w", err)
	}
	// VACUUM INTO cannot run inside a transaction and refuses to write to a path that already exists.
	if _, err := db.Exec("VACUUM INTO ?", tmp); err != nil {
		return "", fmt.Errorf("vacuum into %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, final); err != nil {
		return "", fmt.Errorf("finalize backup %s: %w", final, err)
	}
	return final, nil
}

// PruneBackups deletes every backup in dir except the newest keep, ordered by name (the embedded UTC timestamp
// sorts lexically last, so name order is age order).
func PruneBackups(dir string, keep int) error {
	names, err := backupNames(dir)
	if err != nil {
		return err
	}
	if len(names) <= keep {
		return nil
	}
	for _, n := range names[:len(names)-keep] {
		if err := os.Remove(filepath.Join(dir, n)); err != nil {
			return fmt.Errorf("remove old backup %s: %w", n, err)
		}
	}
	return nil
}

// newestBackupPath returns the newest backup's full path, or "" when dir has none.
func newestBackupPath(dir string) string {
	names, err := backupNames(dir)
	if err != nil || len(names) == 0 {
		return ""
	}
	return filepath.Join(dir, names[len(names)-1])
}

func backupNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), backupPrefix) || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}
