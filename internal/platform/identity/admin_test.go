package identity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeAdmin struct {
	allow bool
	err   error
}

func (f fakeAdmin) CanCreateWorkspace(context.Context, string) (bool, error) { return f.allow, f.err }

func TestRequireInstanceAdmin(t *testing.T) {
	admin := WithActor(context.Background(), Actor{ID: "u1"})
	tests := []struct {
		name    string
		ctx     context.Context
		gate    InstanceAdmin
		wantErr error
	}{
		{"no actor is unauthorized", context.Background(), fakeAdmin{allow: true}, apperrs.ErrUnauthorized},
		{"actor without id is unauthorized", WithActor(context.Background(), Actor{}), fakeAdmin{allow: true}, apperrs.ErrUnauthorized},
		{"no gate is forbidden", admin, nil, apperrs.ErrForbidden},
		{"non-admin is forbidden", admin, fakeAdmin{allow: false}, apperrs.ErrForbidden},
		{"gate failure propagates", admin, fakeAdmin{err: assert.AnError}, assert.AnError},
		{"admin passes", admin, fakeAdmin{allow: true}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireInstanceAdmin(tt.ctx, tt.gate)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
