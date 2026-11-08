package plays

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestParseCron_RefusesWhatCannotFireOrFiresTooOften(t *testing.T) {
	for _, expr := range []string{
		"", "0 23 * *", "60 23 * * *", "0 24 * * *", "0 23 0 * *", "0 23 * 13 *", "0 23 * * 8", "x 23 * * *",
		"0 5-1 * * *", "0 */0 * * *", "0 0 30 2 *", "*/30 * * * *", "* 23 * * *",
	} {
		_, err := parseCron(expr)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid), "%q is refused", expr)
	}
}

func TestCronNext(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	require.NoError(t, err)
	kolkata, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	tests := []struct {
		name string
		expr string
		from time.Time
		want time.Time
	}{
		{"later today", "0 23 * * *", time.Date(2026, 11, 8, 22, 59, 30, 0, london), time.Date(2026, 11, 8, 23, 0, 0, 0, london)},
		{"at the minute is the next day", "0 23 * * *", time.Date(2026, 11, 8, 23, 0, 0, 0, london), time.Date(2026, 11, 9, 23, 0, 0, 0, london)},
		{"weekdays skip the weekend", "30 22 * * mon-fri", time.Date(2026, 11, 6, 23, 0, 0, 0, london), time.Date(2026, 11, 9, 22, 30, 0, 0, london)},
		{"sunday as 7", "0 9 * * 7", time.Date(2026, 11, 8, 10, 0, 0, 0, london), time.Date(2026, 11, 15, 9, 0, 0, 0, london)},
		{"both day fields restricted is either", "0 6 1 * 1", time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 30, 6, 0, 0, 0, time.UTC)},
		{"a list and a stepped hour", "15 1,*/12 * * *", time.Date(2026, 11, 8, 1, 20, 0, 0, time.UTC), time.Date(2026, 11, 8, 12, 15, 0, 0, time.UTC)},
		{"next month by name", "0 0 1 jan *", time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"a half-hour zone keeps its hours", "0 23 * * *", time.Date(2026, 11, 8, 21, 10, 0, 0, kolkata), time.Date(2026, 11, 8, 23, 0, 0, 0, kolkata)},
		{"a skipped spring hour waits a day", "30 1 * * *", time.Date(2026, 3, 29, 0, 0, 0, 0, london), time.Date(2026, 3, 30, 1, 30, 0, 0, london)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseCron(tt.expr)
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(spec.next(tt.from)), "want %s, got %s", tt.want, spec.next(tt.from))
		})
	}
}
