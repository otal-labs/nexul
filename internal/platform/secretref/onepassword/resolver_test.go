package onepassword

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/1password/onepassword-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

type stubSecrets struct {
	resp sdk.ResolveAllResponse
	err  error
}

func (s stubSecrets) Resolve(context.Context, string) (string, error) { return "", nil }
func (s stubSecrets) ResolveAll(context.Context, []string) (sdk.ResolveAllResponse, error) {
	return s.resp, s.err
}

type resolved = sdk.Response[sdk.ResolvedReference, sdk.ResolveReferenceError]

func ok(secret string) resolved {
	return resolved{Content: &sdk.ResolvedReference{Secret: secret}}
}

func fail(e sdk.ResolveReferenceError) resolved { return resolved{Error: &e} }

func TestResolveAll_MapsEachReferenceInOrder(t *testing.T) {
	r := &Resolver{secrets: stubSecrets{resp: sdk.ResolveAllResponse{IndividualResponses: map[string]resolved{
		"op://v/good/f":    ok("s3cret"),
		"op://v/missing/f": fail(sdk.NewResolveReferenceErrorTypeVariantItemNotFound()),
		"op://v/item/nope": fail(sdk.NewResolveReferenceErrorTypeVariantFieldNotFound()),
		"op://locked/i/f":  fail(sdk.NewResolveReferenceErrorTypeVariantVaultNotFound()),
		"op://v/dup/f":     fail(sdk.NewResolveReferenceErrorTypeVariantTooManyItems()),
		"op://v/empty/f":   {},
	}}}}
	refs := []string{"op://v/missing/f", "op://v/good/f", "op://locked/i/f", "op://v/item/nope", "op://v/dup/f", "op://v/empty/f", "op://v/unanswered/f"}

	got, err := r.ResolveAll(t.Context(), refs)

	require.NoError(t, err)
	require.Len(t, got, len(refs))
	assert.ErrorIs(t, got[0].Err, secretref.ErrNotFound)
	assert.Equal(t, secretref.Result{Value: "s3cret"}, got[1])
	assert.ErrorIs(t, got[2].Err, secretref.ErrNoAccess)
	assert.ErrorIs(t, got[3].Err, secretref.ErrFieldMissing)
	assert.ErrorIs(t, got[4].Err, secretref.ErrFailed)
	assert.ErrorIs(t, got[5].Err, secretref.ErrFailed)
	assert.ErrorIs(t, got[6].Err, secretref.ErrFailed)
}

func TestResolveAll_ClassifiesACallFailure(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	tests := []struct {
		name        string
		ctx         context.Context
		err         error
		unreachable bool
	}{
		{"dial error", t.Context(), errors.New(`dial tcp: lookup my.1password.com: no such host`), true},
		{"other failure", t.Context(), errors.New("invalid service account token"), false},
		{"cancelled call", cancelled, errors.New("dial tcp: operation was canceled"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Resolver{secrets: stubSecrets{err: tt.err}}

			got, err := r.ResolveAll(tt.ctx, []string{"op://v/i/f"})

			require.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, tt.unreachable, errors.Is(err, secretref.ErrUnreachable))
		})
	}
}
