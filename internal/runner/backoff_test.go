package runner

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackoff_Delay(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	b := NewBackoff(time.Second, 8*time.Second, rng)

	first := b.Delay(0)
	assert.GreaterOrEqual(t, first, time.Second)
	assert.LessOrEqual(t, first, 2*time.Second)

	assert.GreaterOrEqual(t, b.Delay(1), 2*time.Second)
	assert.LessOrEqual(t, b.Delay(1), 4*time.Second)

	t.Run("caps at max", func(t *testing.T) {
		got := b.Delay(100)
		assert.GreaterOrEqual(t, got, 8*time.Second)
		assert.LessOrEqual(t, got, 12*time.Second)
	})

	t.Run("nil rng is deterministic", func(t *testing.T) {
		nb := NewBackoff(time.Second, time.Minute, nil)
		assert.Equal(t, nb.Delay(2), nb.Delay(2))
		assert.Equal(t, 4*time.Second, nb.Delay(2))
	})
}

func TestBackoff_Wait_Cancelled(t *testing.T) {
	b := NewBackoff(time.Hour, time.Hour, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	ok := b.Wait(ctx, 0)
	assert.False(t, ok)
	assert.Less(t, time.Since(start), 50*time.Millisecond)
}

func TestBackoff_Wait_Expires(t *testing.T) {
	b := NewBackoff(time.Millisecond, time.Millisecond, nil)
	start := time.Now()
	ok := b.Wait(context.Background(), 0)
	assert.True(t, ok)
	require.True(t, time.Since(start) >= time.Millisecond)
}
