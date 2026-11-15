package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiryOf_IsMidnightUTCOnTheDateAndRefusesAKeyAlreadyExpired(t *testing.T) {
	t.Parallel()
	// Late on 16 Nov UTC: today is the UTC date, and its midnight has passed.
	now := time.Date(2026, 11, 16, 23, 30, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		expires string
		days    int
		want    string
		wantErr string
	}{
		{"a monthly key with its grace", "", 45, "2026-12-31T00:00:00Z", ""},
		{"a pilot of 90 days", "", 90, "2027-02-14T00:00:00Z", ""},
		{"a date", "2027-11-11", 0, "2027-11-11T00:00:00Z", ""},
		{"tomorrow", "2026-11-17", 0, "2026-11-17T00:00:00Z", ""},
		{"today, which has begun", "2026-11-16", 0, "", "already begun"},
		{"both", "2027-11-11", 30, "", "one of"},
		{"neither", "", 0, "", "one of"},
		{"negative days", "", -1, "", "already begun"},
		{"not a date", "11/11/2027", 0, "", "not a YYYY-MM-DD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := expiryOf(tc.expires, tc.days, now)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Format(time.RFC3339))
		})
	}
}
