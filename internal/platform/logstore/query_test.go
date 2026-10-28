package logstore

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var now = time.Date(2026, 9, 25, 12, 0, 0, 123456789, time.FixedZone("x", 3600))

func TestParams_Query_RefusesWhatTheTableDoesNotAllow(t *testing.T) {
	tests := []struct {
		name string
		p    Params
		want string
	}{
		{"unknown source", Params{Source: "database"}, "source must be one of"},
		{"unknown level", Params{Level: "fatal"}, "level must be one of"},
		{"since that is no time", Params{Since: "yesterday"}, "since must be a time"},
		{"a negative duration", Params{Since: "-5m"}, "since must be a time"},
		{"days past the cap are refused before they overflow", Params{Since: "999999999999d"}, "since must be a time"},
		{"until that is no time", Params{Until: "later"}, "until must be a time"},
		{"a range past 7 days", Params{Since: "8d"}, "at most 7 days"},
		{"since after until", Params{Since: "2026-09-25T13:00:00Z", Until: "2026-09-25T10:00:00Z"}, "before until"},
		{"a backslash", Params{Text: `a\b`}, "backslash"},
		{"a newline", Params{TraceID: "a\nb"}, "control character"},
		{"a nul byte", Params{Name: "a\x00b"}, "control character"},
		{"a value too long", Params{Text: strings.Repeat("a", maxValueLen+1)}, "at most 200"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.p.Query(now)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestParams_Query_ResolvesDefaultsAndPinsUntil(t *testing.T) {
	q, err := Params{}.Query(now)
	require.NoError(t, err)
	assert.Equal(t, "info", q.Level)
	assert.Equal(t, time.Date(2026, 9, 25, 11, 0, 0, 123456000, time.UTC), q.Until, "now in UTC, to the microsecond")
	assert.Equal(t, q.Until.Add(-time.Hour), q.Since)
	assert.Equal(t, 100, q.Window.Limit)
	assert.Equal(t, 0, q.Window.Offset)
}

func TestParams_Query_Moments(t *testing.T) {
	tests := []struct {
		name         string
		p            Params
		since, until time.Time
	}{
		{"a duration back from now", Params{Since: "30m"}, now.UTC().Truncate(time.Microsecond).Add(-30 * time.Minute), now.UTC().Truncate(time.Microsecond)},
		{"whole days", Params{Since: "2d"}, now.UTC().Truncate(time.Microsecond).Add(-48 * time.Hour), now.UTC().Truncate(time.Microsecond)},
		{"exactly seven days", Params{Since: "7d"}, now.UTC().Truncate(time.Microsecond).Add(-7 * 24 * time.Hour), now.UTC().Truncate(time.Microsecond)},
		{"times", Params{Since: "2026-09-25T09:00:00Z", Until: "2026-09-25T10:00:00Z"}, time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC), time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)},
		{"a pinned until with the default hour before it", Params{Until: "2026-09-25T10:00:00.5Z"}, time.Date(2026, 9, 25, 9, 0, 0, 500000000, time.UTC), time.Date(2026, 9, 25, 10, 0, 0, 500000000, time.UTC)},
		{"a duration as until counts back from now", Params{Until: "10m", Since: "1h"}, now.UTC().Truncate(time.Microsecond).Add(-time.Hour - 10*time.Minute), now.UTC().Truncate(time.Microsecond).Add(-10 * time.Minute)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := tt.p.Query(now)
			require.NoError(t, err)
			assert.Equal(t, tt.since, q.Since)
			assert.Equal(t, tt.until, q.Until)
		})
	}
}

func TestParams_Query_Window(t *testing.T) {
	tests := []struct {
		name          string
		p             Params
		limit, offset int
	}{
		{"a limit inside the range", Params{Limit: 7, Offset: 14}, 7, 14},
		{"a limit past the cap", Params{Limit: 9999}, 500, 0},
		{"a negative limit", Params{Limit: -3}, 100, 0},
		{"a negative offset", Params{Offset: -5}, 100, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := tt.p.Query(now)
			require.NoError(t, err)
			assert.Equal(t, tt.limit, q.Window.Limit)
			assert.Equal(t, tt.offset, q.Window.Offset)
		})
	}
}

func TestParams_Query_LevelIsCaseInsensitive(t *testing.T) {
	q, err := Params{Level: "WARN"}.Query(now)
	require.NoError(t, err)
	assert.Equal(t, "warn", q.Level)
}

func TestParamsOf(t *testing.T) {
	t.Run("reads every field", func(t *testing.T) {
		v := url.Values{"source": {"web"}, "level": {"warn"}, "since": {"2h"}, "until": {"2026-09-25T10:00:00Z"},
			"trace_id": {"t1"}, "name": {"edge"}, "text": {"boom"}, "limit": {"20"}, "offset": {"40"}}
		p, err := ParamsOf(v)
		require.NoError(t, err)
		assert.Equal(t, Params{Source: "web", Level: "warn", Since: "2h", Until: "2026-09-25T10:00:00Z", TraceID: "t1", Name: "edge", Text: "boom", Limit: 20, Offset: 40}, p)
	})

	for _, key := range []string{"limit", "offset"} {
		t.Run("a "+key+" that is no number", func(t *testing.T) {
			_, err := ParamsOf(url.Values{key: {"many"}})
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, key+" must be a whole number")
		})
	}
}
