package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// WAL gives concurrent readers, synchronous(NORMAL) is safe under it, and busy_timeout waits on a writer only because transactions begin immediate.
const pragmas = "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_txlock=immediate"

func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?"+pragmas)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// A single connection turns one stuck statement into an outage; the lifetime cap recycles stale WAL snapshots.
	db.SetMaxOpenConns(8)
	// Opening a connection re-parses the whole schema, so a burst must never close the ones it opened.
	db.SetMaxIdleConns(8)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// idsJSON renders ids as a JSON array, the form a query reads a runtime set in through json_each, so one static
// statement serves any number of ids and may name the set twice.
func idsJSON(ids []string) string {
	if ids == nil {
		ids = []string{}
	}
	b, _ := json.Marshal(ids) // a []string always encodes
	return string(b)
}
