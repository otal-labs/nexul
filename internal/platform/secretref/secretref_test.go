package secretref

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		value string
		kind  Kind
		ref   Ref
	}{
		{"three parts", "op://acme/db/password", Reference, Ref{Scheme: "op", Vault: "acme", Item: "db", Field: "password"}},
		{"with section", "op://acme/db/prod/password", Reference, Ref{Scheme: "op", Vault: "acme", Item: "db", Section: "prod", Field: "password"}},
		{"spaces in names", "op://Acme Ops/Prod DB/password", Reference, Ref{Scheme: "op", Vault: "Acme Ops", Item: "Prod DB", Field: "password"}},
		{"postgres url", "postgres://u:p@example.com/db", Literal, Ref{}},
		{"https url", "https://example.com/a/b/c", Literal, Ref{}},
		{"embedded", "x op://acme/db/password", Literal, Ref{}},
		{"plain value", "hunter2", Literal, Ref{}},
		{"empty", "", Literal, Ref{}},
		{"vault only", "op://acme", Malformed, Ref{}},
		{"two parts", "op://acme/db", Malformed, Ref{}},
		{"empty vault", "op:///db/password", Malformed, Ref{}},
		{"empty field", "op://acme/db/", Malformed, Ref{}},
		{"too many parts", "op://a/b/c/d/e", Malformed, Ref{}},
		{"trailing newline", "op://acme/db/password\n", Malformed, Ref{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ref := Parse(tt.value)
			assert.Equal(t, tt.kind, kind)
			assert.Equal(t, tt.ref, ref)
		})
	}
}

func TestCheck_NamesTheHolderAndNeverTheValue(t *testing.T) {
	require.NoError(t, Check("DB_URL", "op://acme/db/password"))
	require.NoError(t, Check("DB_URL", "literal"))

	err := Check("DB_URL", "op://topsecretvault")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "DB_URL")
	assert.NotContains(t, err.Error(), "topsecretvault")
}
