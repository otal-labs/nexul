package plays

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestFiring_SweepsTheLatestMissedFiringWhileItCanStillStartSomething(t *testing.T) {
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 11, day, hour, minute, 0, 0, time.UTC) }
	nightly := func(stopAt string, updated time.Time) *AutoPlay {
		return &AutoPlay{Moment: MomentSchedule, UpdatedAt: updated, Schedule: &Schedule{Cron: "0 23 * * *", Timezone: "UTC", StopAt: stopAt, MaxRuns: 10}}
	}
	tests := []struct {
		name     string
		a        *AutoPlay
		last     *Sweep
		now      time.Time
		due      time.Time
		upcoming time.Time
	}{
		{"not yet", nightly("", at(1, 9, 0)), nil, at(8, 22, 59), time.Time{}, at(8, 23, 0)},
		{"on time", nightly("", at(1, 9, 0)), &Sweep{FiredAt: at(7, 23, 0)}, at(8, 23, 0), at(8, 23, 0), at(9, 23, 0)},
		{"already swept", nightly("", at(1, 9, 0)), &Sweep{FiredAt: at(8, 23, 0)}, at(8, 23, 30), time.Time{}, at(9, 23, 0)},
		{"late within the hour", nightly("", at(1, 9, 0)), &Sweep{FiredAt: at(6, 23, 0)}, at(8, 23, 40), at(8, 23, 0), at(9, 23, 0)},
		{"too late without a stop", nightly("", at(1, 9, 0)), nil, at(9, 1, 0), time.Time{}, at(9, 23, 0)},
		{"late but before its stop", nightly("07:00", at(1, 9, 0)), nil, at(9, 5, 0), at(8, 23, 0), at(9, 23, 0)},
		{"after its stop", nightly("07:00", at(1, 9, 0)), nil, at(9, 7, 0), time.Time{}, at(9, 23, 0)},
		{"an edit starts over from itself", nightly("", at(8, 23, 30)), &Sweep{FiredAt: at(7, 23, 0)}, at(8, 23, 40), time.Time{}, at(9, 23, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			due, upcoming, ok := firing(tt.a, tt.last, tt.now)
			assert.True(t, ok)
			assert.True(t, tt.due.Equal(due), "due: want %s, got %s", tt.due, due)
			assert.True(t, tt.upcoming.Equal(upcoming), "upcoming: want %s, got %s", tt.upcoming, upcoming)
		})
	}
}

func TestAutoPlay_Validate_Schedule(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(*AutoPlay)
		valid bool
	}{
		{"defaults fill in", func(*AutoPlay) {}, true},
		{"a causer is refused", func(a *AutoPlay) { a.RunOn = RunOnCauser }, false},
		{"an unknown zone", func(a *AutoPlay) { a.Schedule.Timezone = "Mars/Olympus" }, false},
		{"a stop that is no clock", func(a *AutoPlay) { a.Schedule.StopAt = "7am" }, false},
		{"too many runs", func(a *AutoPlay) { a.Schedule.MaxRuns = MaxSweepRuns + 1 }, false},
		{"too often", func(a *AutoPlay) { a.Schedule.Cron = "*/15 * * * *" }, false},
		{"a long resource", func(a *AutoPlay) { a.OneAtATime, a.Resource = true, "a-name-far-longer-than-forty-characters-long" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AutoPlay{Moment: MomentSchedule, Schedule: &Schedule{Cron: "0 23 * * *"}}
			a.normalize(TypeTicket)
			tt.edit(a)
			err := a.Validate(TypeTicket)
			if tt.valid {
				assert.NoError(t, err)
				assert.Equal(t, "UTC", a.Schedule.Timezone)
				assert.Equal(t, DefaultSweepRuns, a.Schedule.MaxRuns)
				return
			}
			assert.True(t, errors.Is(err, apperrs.ErrInvalid), "%v", err)
		})
	}
}

func TestAutoPlay_Normalize_DropsAScheduleAndAResourceNothingUses(t *testing.T) {
	a := &AutoPlay{Moment: MomentTicketCreated, Schedule: &Schedule{Cron: "0 23 * * *"}, Resource: "windows-vm"}
	a.normalize(TypeTicket)
	assert.Nil(t, a.Schedule)
	assert.Empty(t, a.Resource)
	assert.Empty(t, a.LockKey())
}
