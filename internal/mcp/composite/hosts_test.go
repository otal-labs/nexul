package composite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeHostKind records what the host tools asked of one kind.
type fakeHostKind struct {
	enrolled [][2]string
	removed  []string
	revoked  []string
	err      error
}

// noCodesKind is a host kind that cannot revoke codes.
type noCodesKind struct{}

func (noCodesKind) Enroll(context.Context, string, string) (any, error) { return nil, nil }
func (noCodesKind) Remove(context.Context, string) error                { return nil }

func (f *fakeHostKind) RevokeCode(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.revoked = append(f.revoked, id)
	return nil
}

func (f *fakeHostKind) Enroll(_ context.Context, name, machine string) (any, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.enrolled = append(f.enrolled, [2]string{name, machine})
	return map[string]string{"code": "nxe_1"}, nil
}

func (f *fakeHostKind) Remove(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	f.removed = append(f.removed, id)
	return nil
}

func TestHostTools_Refusals(t *testing.T) {
	tests := []struct {
		name string
		tool string
		args string
		err  error
		want error
		msg  string
	}{
		{"a kind nobody wired yet", "host_create", `{"kind":"automations","name":"a"}`, nil, apperrs.ErrInvalid, "use one of: runner"},
		{"an unknown kind", "host_delete", `{"kind":"printer","id":"r-1"}`, nil, apperrs.ErrInvalid, "use one of: runner"},
		{"create needs a name", "host_create", `{"kind":"runner"}`, nil, apperrs.ErrInvalid, ""},
		{"delete needs an id", "host_delete", `{"kind":"runner"}`, nil, apperrs.ErrInvalid, ""},
		{"an id and a code id together are invalid", "host_delete", `{"kind":"runner","id":"r-1","code_id":"c-1"}`, nil, apperrs.ErrInvalid, "either id"},
		{"a missing code names the list tool", "host_delete", `{"kind":"runner","code_id":"ghost"}`, apperrs.ErrNotFound, apperrs.ErrNotFound, "machine_list"},
		{"the kind's refusal passes through", "host_create", `{"kind":"runner","name":"a"}`, apperrs.ErrForbidden, apperrs.ErrForbidden, ""},
		{"a missing host names the list tool", "host_delete", `{"kind":"runner","id":"ghost"}`, apperrs.ErrNotFound, apperrs.ErrNotFound, "machine_list"},
		{"a failed removal passes through", "host_delete", `{"kind":"runner","id":"r-1"}`, apperrs.ErrForbidden, apperrs.ErrForbidden, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := HostTools(map[string]HostKind{"runner": &fakeHostKind{err: tt.err}})
			_, err := call(t, context.Background(), tools, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}

func TestHostTools_DispatchToTheKind(t *testing.T) {
	runner := &fakeHostKind{}
	tools := HostTools(map[string]HostKind{"runner": runner})

	out, err := call(t, context.Background(), tools, "host_create", `{"kind":"runner","name":"build-box","machine":"prod"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"code": "nxe_1"}, out)
	assert.Equal(t, [][2]string{{"build-box", "prod"}}, runner.enrolled)

	out, err = call(t, context.Background(), tools, "host_delete", `{"kind":"runner","id":"r-1"}`)
	require.NoError(t, err)
	assert.Equal(t, hostDeleted{ID: "r-1", Deleted: true}, out)
	assert.Equal(t, []string{"r-1"}, runner.removed)
}

func TestHostDelete_ARevokeNeedsAKindThatHasCodes(t *testing.T) {
	tools := HostTools(map[string]HostKind{"automations": noCodesKind{}})
	_, err := call(t, context.Background(), tools, "host_delete", `{"kind":"automations","code_id":"c-1"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "only runner codes")
}

func TestHostDelete_CodeIDRevokesTheCodeAndRemovesNoHost(t *testing.T) {
	runner := &fakeHostKind{}
	out, err := call(t, context.Background(), HostTools(map[string]HostKind{"runner": runner}), "host_delete", `{"kind":"runner","code_id":"c-1"}`)
	require.NoError(t, err)
	assert.Equal(t, hostDeleted{ID: "c-1", Deleted: true}, out)
	assert.Equal(t, []string{"c-1"}, runner.revoked)
	assert.Empty(t, runner.removed)
}
