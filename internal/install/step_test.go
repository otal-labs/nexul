package install

import (
	"bytes"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStep_ElapsedTime(t *testing.T) {
	line := "  Docker .......... "
	tests := []struct {
		name       string
		live       bool
		run        func(h *Host) (string, error)
		wantScreen string
		wantShown  string
	}{
		{
			name:       "slow step on a terminal shows its elapsed time, then the result over it",
			live:       true,
			run:        func(*Host) (string, error) { time.Sleep(12500 * time.Millisecond); return "29.0", nil },
			wantScreen: line + "29.0\n",
			wantShown:  line + "12s",
		},
		{
			name:       "fast step draws nothing extra",
			live:       true,
			run:        func(*Host) (string, error) { return "29.0", nil },
			wantScreen: line + "29.0\n",
		},
		{
			name:       "no terminal, no redraw",
			live:       false,
			run:        func(*Host) (string, error) { time.Sleep(3 * time.Second); return "29.0", nil },
			wantScreen: line + "29.0\n",
		},
		{
			name: "output during the step stops the redraw",
			live: true,
			run: func(h *Host) (string, error) {
				time.Sleep(1500 * time.Millisecond)
				h.printf("\n  Installing Homebrew; it asks for your password.\n")
				time.Sleep(3 * time.Second)
				return "29.0", nil
			},
			wantScreen: line + "1s\n  Installing Homebrew; it asks for your password.\n29.0\n",
			wantShown:  line + "1s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				out := &bytes.Buffer{}
				h := &Host{Out: out, Live: tt.live}
				require.NoError(t, h.step("Docker", func() (string, error) { return tt.run(h) }))
				assert.Equal(t, tt.wantScreen, screen(out.String()))
				if tt.wantShown == "" {
					assert.NotContains(t, out.String(), "\r")
					return
				}
				assert.Contains(t, out.String(), tt.wantShown)
			})
		})
	}
}

// screen is what a terminal shows for s: each carriage return rewrites the line from its start.
func screen(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		var cur []byte
		for _, part := range strings.Split(l, "\r") {
			cur = append([]byte(part), cur[min(len(part), len(cur)):]...)
		}
		lines[i] = strings.TrimRight(string(cur), " ")
	}
	return strings.Join(lines, "\n")
}
