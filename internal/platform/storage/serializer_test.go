package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
)

func TestSerializer_CommitsOnSuccess(t *testing.T) {
	db := newTestDB(t)

	w := &Serializer{}
	err := w.WithTx(context.Background(), db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(context.Background(),
			`INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			"d1", "title", "body", 1, 1, 1)
		return err
	})
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM docs`).Scan(&n))
	assert.Equal(t, 1, n)
}

func TestSerializer_RollsBackOnError(t *testing.T) {
	db := newTestDB(t)

	w := &Serializer{}
	err := w.WithTx(context.Background(), db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(context.Background(),
			`INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			"d1", "title", "body", 1, 1, 1); err != nil {
			return err
		}
		return fmt.Errorf("boom")
	})
	require.Error(t, err)

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM docs`).Scan(&n))
	assert.Equal(t, 0, n, "partial write rolled back")
}

func TestSerializer_WithTx_ClosedDB_Error(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Close())

	w := &Serializer{}
	err := w.WithTx(context.Background(), db, func(tx *sql.Tx) error {
		return nil
	})
	require.Error(t, err)
}

func TestSerializer_ConcurrentWrites(t *testing.T) {
	s := newTestStore(t)

	const workers = 20
	var wg sync.WaitGroup
	errs := make([]error, workers)
	now := time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := &docs.Doc{ID: fmt.Sprintf("doc-%d", i), Title: "t", Body: "b", Version: 1, CreatedAt: now, UpdatedAt: now}
			errs[i] = s.Docs.Create(context.Background(), d)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "worker %d", i)
	}
	got, err := s.Docs.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, workers)
}
