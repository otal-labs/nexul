package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func editMemory(t *testing.T, s *Store, m *memories.Memory, edit func(*memories.Memory)) {
	t.Helper()
	m.UpdatedAt = m.UpdatedAt.Add(time.Minute)
	edit(m)
	require.NoError(t, s.Memories.Update(t.Context(), m, ""))
}

func memoryVersionNumbers(t *testing.T, s *Store, id string) []int {
	t.Helper()
	vs, err := s.Memories.ListVersionSummaries(t.Context(), id)
	require.NoError(t, err)
	var out []int
	for _, v := range vs {
		out = append(out, v.Version)
	}
	return out
}

func TestMemoriesRepo_Update_ASaveThatChangesNothingWritesNoVersion(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(t.Context(), m, ""))

	editMemory(t, s, m, func(*memories.Memory) {})
	editMemory(t, s, m, func(m *memories.Memory) { m.Footer = true })
	assert.Equal(t, []int{1}, memoryVersionNumbers(t, s, "mem-1"), "a save, or a footer flag, that leaves the four versioned fields as they were adds no row")

	changes := map[string]func(*memories.Memory){
		"title":           func(m *memories.Memory) { m.Title = "Renamed" },
		"when to use":     func(m *memories.Memory) { m.WhenToUse = "another hint" },
		"body":            func(m *memories.Memory) { m.Body = "Other body" },
		"always included": func(m *memories.Memory) { m.AlwaysIncluded = !m.AlwaysIncluded },
	}
	for name, change := range changes {
		before := len(memoryVersionNumbers(t, s, "mem-1"))
		editMemory(t, s, m, change)
		assert.Len(t, memoryVersionNumbers(t, s, "mem-1"), before+1, "changing the %s adds a version", name)
	}
}

func TestMemoriesRepo_Update_KeepsTheNewestTenAndSizesTheOldestAsNone(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(t.Context(), m, ""))
	for i := 2; i <= 12; i++ {
		editMemory(t, s, m, func(m *memories.Memory) { m.Body = fmt.Sprintf("v%d only", i) })
	}

	h, err := s.Memories.ListVersionSummaries(t.Context(), "mem-1")
	require.NoError(t, err)
	require.Len(t, h, memories.KeptVersions)
	assert.Equal(t, 12, h[0].Version)
	assert.Equal(t, "+1 -1", sizeLabel(h[0].WordsAdded, h[0].WordsRemoved), "v12 swaps one word of v11")
	assert.Equal(t, 3, h[9].Version)
	assert.Nil(t, h[9].WordsAdded, "version 3 lost its predecessor to the prune")
	assert.Nil(t, h[9].WordsRemoved)

	_, err = s.Memories.GetVersion(t.Context(), "mem-1", 2)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMemoriesRepo_Update_ARestoreNamesTheVersionItCameFromAndAPlainSaveLeavesItNull(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(t.Context(), m, ""))
	editMemory(t, s, m, func(m *memories.Memory) { m.Body = "second" })
	editMemory(t, s, m, func(m *memories.Memory) { m.Body, m.RestoredFrom = "Body", 1 })

	h, err := s.Memories.ListVersionSummaries(t.Context(), "mem-1")
	require.NoError(t, err)
	require.Len(t, h, 3)
	require.NotNil(t, h[0].RestoredFrom)
	assert.Equal(t, 1, *h[0].RestoredFrom)
	assert.Nil(t, h[1].RestoredFrom)
	assert.Nil(t, h[2].RestoredFrom)
}
