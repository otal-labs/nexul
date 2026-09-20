package automations

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestAutomation_Validate(t *testing.T) {
	tests := []struct {
		name    string
		a       Automation
		wantErr bool
	}{
		{"valid custom", Automation{Name: "My automation", Kind: KindCustom}, false},
		{"valid default", Automation{Name: "Shipped automation", Kind: KindDefault}, false},
		{"empty name", Automation{Name: "  ", Kind: KindCustom}, true},
		{"unknown kind", Automation{Name: "x", Kind: "bogus"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.a.Validate()
			if tt.wantErr {
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestNormalizeScopes(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr bool
	}{
		{"trims and dedupes preserving order", []string{" tickets:write ", "tickets:write", "docs:read"}, []string{"tickets:write", "docs:read"}, false},
		{"drops blanks", []string{"", "  ", "deploys:read"}, []string{"deploys:read"}, false},
		{"empty is invalid", nil, nil, true},
		{"all blank is invalid", []string{"", "  "}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeScopes(tt.in)
			if tt.wantErr {
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
