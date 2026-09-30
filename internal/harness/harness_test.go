package harness

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestCapDetail_CutsOnARuneBoundary(t *testing.T) {
	long := strings.Repeat("é", MaxActivityDetail)
	got := CapDetail(long)
	assert.LessOrEqual(t, len(got), MaxActivityDetail)
	assert.True(t, strings.HasSuffix(got, "é"), "never ends mid-rune")
	assert.Equal(t, "short", CapDetail("short"))
}

func TestPreview_FlattensAndTruncates(t *testing.T) {
	assert.Equal(t, "a b c", Preview("a\n  b\t c", 10))
	assert.Equal(t, "abc…", Preview("abcdef", 3))
	assert.Equal(t, "", Preview("  \n ", 5))
}

func TestCleanOptions(t *testing.T) {
	tests := []struct {
		name    string
		in      []OptionSetting
		want    []OptionSetting
		wantErr bool
	}{
		{"none", nil, nil, false},
		{"choice and switch trimmed", []OptionSetting{{ID: " effort ", Value: " high "}, {ID: "fastMode", Value: false}},
			[]OptionSetting{{ID: "effort", Value: "high"}, {ID: "fastMode", Value: false}}, false},
		{"missing id", []OptionSetting{{ID: " ", Value: "high"}}, nil, true},
		{"blank choice", []OptionSetting{{ID: "effort", Value: "  "}}, nil, true},
		{"set twice", []OptionSetting{{ID: "effort", Value: "low"}, {ID: "effort", Value: "high"}}, nil, true},
		{"number", []OptionSetting{{ID: "effort", Value: 3.0}}, nil, true},
		{"object", []OptionSetting{{ID: "effort", Value: map[string]any{}}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanOptions(tt.in)
			if tt.wantErr {
				assert.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
