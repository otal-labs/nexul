package inprocess

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackoff_Delay_ExponentialWithJitter(t *testing.T) {
	b := NewBackoff(100*time.Millisecond, 30*time.Second, rand.New(rand.NewSource(42)))
	tests := []struct {
		name    string
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{"first failure", 0, 100 * time.Millisecond, 150 * time.Millisecond},
		{"second failure", 1, 200 * time.Millisecond, 300 * time.Millisecond},
		{"third failure", 2, 400 * time.Millisecond, 600 * time.Millisecond},
		{"capped at max", 10, 30 * time.Second, 45 * time.Second},
		{"capped at max deep", 64, 30 * time.Second, 45 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.Delay(tt.attempt)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

func TestBackoff_Delay_NoRngIsPure(t *testing.T) {
	b := NewBackoff(10*time.Millisecond, time.Hour, nil)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 10 * time.Millisecond},
		{1, 20 * time.Millisecond},
		{2, 40 * time.Millisecond},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, b.Delay(tt.attempt))
	}
}

func TestBackoff_Delay_ZeroInitialIsZero(t *testing.T) {
	b := NewBackoff(0, time.Second, rand.New(rand.NewSource(1)))
	require.Zero(t, b.Delay(3))
}
