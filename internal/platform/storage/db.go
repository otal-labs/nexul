package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// WAL gives concurrent readers, busy_timeout absorbs lock contention, and synchronous(NORMAL) is safe under WAL while being faster than FULL.
const pragmas = "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?"+pragmas)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// A single connection turns one stuck statement into an outage; the lifetime cap recycles stale WAL snapshots.
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
