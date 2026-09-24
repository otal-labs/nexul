package mcptool

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestRequiredString(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		key     string
		want    string
		wantErr error
	}{
		{name: "present", args: map[string]any{"id": "t-1"}, key: "id", want: "t-1"},
		{name: "missing", args: map[string]any{}, key: "id", wantErr: apperrs.ErrInvalid},
		{
			name:    "wrong type",
			args:    map[string]any{"id": 7},
			key:     "id",
			wantErr: apperrs.ErrInvalid,
		},
		{
			name:    "empty",
			args:    map[string]any{"id": ""},
			key:     "id",
			wantErr: apperrs.ErrInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequiredString(tt.args, tt.key)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("RequiredString() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RequiredString() unexpected error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("RequiredString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequiredStrings(t *testing.T) {
	t.Run("all present", func(t *testing.T) {
		got, err := RequiredStrings(map[string]any{"project_id": "p-1", "title": "x"}, "project_id", "title")
		if err != nil {
			t.Fatalf("RequiredStrings() unexpected error = %v", err)
		}
		want := []string{"p-1", "x"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("RequiredStrings() = %v, want %v", got, want)
		}
	})

	t.Run("short-circuits on first missing", func(t *testing.T) {
		_, err := RequiredStrings(map[string]any{"project_id": "p-1"}, "project_id", "title")
		if !errors.Is(err, apperrs.ErrInvalid) {
			t.Fatalf("RequiredStrings() error = %v, want %v", err, apperrs.ErrInvalid)
		}
	})
}

func TestOptionalString(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{name: "string", v: "main", want: "main"},
		{name: "empty string", v: "", want: ""},
		{name: "missing", v: nil, want: ""},
		{name: "number", v: float64(3), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OptionalString(tt.v); got != tt.want {
				t.Fatalf("OptionalString(%v) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestObjectSchema(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		required   []string
		want       map[string]any
	}{
		{"nil properties become an empty object", nil, nil, map[string]any{"type": "object", "properties": map[string]any{}}},
		{"required is kept", map[string]any{"id": map[string]any{"type": "string"}}, []string{"id"},
			map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjectSchema(tt.properties, tt.required...)
			assert.Equal(t, tt.want, got)
			raw, err := json.Marshal(got)
			require.NoError(t, err)
			assert.NotContains(t, string(raw), "null")
		})
	}
}
