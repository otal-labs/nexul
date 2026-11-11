package secretref_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func TestRegistry_For(t *testing.T) {
	op := &secretreftest.Fake{}
	other := &secretreftest.Fake{SchemeName: "vault"}
	reg := secretref.NewRegistry(op, other)

	tests := []struct {
		name  string
		value string
		want  secretref.Resolver
		err   bool
	}{
		{"op reference", "op://v/i/f", op, false},
		{"literal", "hunter2", nil, true},
		{"malformed reference", "op://v/i", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reg.For(tt.value)
			if tt.err {
				require.ErrorIs(t, err, secretref.ErrNoResolver)
				assert.NotContains(t, err.Error(), tt.value)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tt.want, got)
		})
	}

	_, err := secretref.NewRegistry(other).For("op://v/i/f")
	require.ErrorIs(t, err, secretref.ErrNoResolver)
}

func TestFake_BatchOfGoodAndBadEntries(t *testing.T) {
	boom := errors.New("boom")
	f := &secretreftest.Fake{
		Values: map[string]string{"op://v/a/f": "one", "op://v/b/f": "shadowed"},
		Errs: map[string]error{
			"op://v/b/f": secretref.ErrNoAccess,
			"op://v/c/f": secretref.ErrFieldMissing,
			"op://v/d/f": secretref.ErrUnreachable,
		},
	}
	refs := []string{"op://v/a/f", "op://v/b/f", "op://v/c/f", "op://v/d/f", "op://v/none/f"}

	got, err := f.ResolveAll(t.Context(), refs)

	require.NoError(t, err)
	assert.Equal(t, secretref.Result{Value: "one"}, got[0])
	assert.ErrorIs(t, got[1].Err, secretref.ErrNoAccess)
	assert.ErrorIs(t, got[2].Err, secretref.ErrFieldMissing)
	assert.ErrorIs(t, got[3].Err, secretref.ErrUnreachable)
	assert.ErrorIs(t, got[4].Err, secretref.ErrNotFound)
	assert.Equal(t, [][]string{refs}, f.Calls())

	f.CallErr = boom
	_, err = f.ResolveAll(t.Context(), refs)
	assert.ErrorIs(t, err, boom)
}
