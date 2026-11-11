package secretref_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

const cfRef = "op://Acme/cf/token"

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

// cacheOver opens one manager whose open step is open, the way the server does at start.
func cacheOver(t *testing.T, open func(context.Context) (secretref.Resolver, error)) (*secretref.Cache, *clock) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	managers := secretref.NewManagers(slog.New(slog.NewTextHandler(io.Discard, nil)), secretref.Setup{Scheme: "op", Open: open})
	managers.SetRetry(time.Hour, time.Hour)
	var wg sync.WaitGroup
	managers.Start(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	clk := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return secretref.NewCache(managers, clk.now), clk
}

func opening(res secretref.Resolver) func(context.Context) (secretref.Resolver, error) {
	return func(context.Context) (secretref.Resolver, error) { return res, nil }
}

func TestCache_ReadsAReferenceOnceAndPassesALiteralThrough(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{cfRef: "tok-1"}}
	cache, _ := cacheOver(t, opening(fake))

	for range 3 {
		got, err := cache.Value(t.Context(), cfRef)
		require.NoError(t, err)
		assert.Equal(t, "tok-1", got)
	}
	literal, err := cache.Value(t.Context(), "plain-token")
	require.NoError(t, err)
	assert.Equal(t, "plain-token", literal)

	assert.Len(t, fake.Calls(), 1, "a resolved value stays in memory")
}

func TestCache_AFailingReferenceIsReadAtMostAboutAHundredTimesADay(t *testing.T) {
	fake := &secretreftest.Fake{Errs: map[string]error{cfRef: secretref.ErrUnreachable}}
	cache, clk := cacheOver(t, opening(fake))

	for range 24 * 60 * 2 {
		_, err := cache.Value(t.Context(), cfRef)
		require.ErrorIs(t, err, secretref.ErrUnreachable)
		clk.advance(30 * time.Second)
	}

	assert.InDelta(t, 100, len(fake.Calls()), 8)
}

func TestCache_AManagerThatDidNotOpen_AnswersWithWhyAndReadsNothing(t *testing.T) {
	tests := []struct {
		name    string
		openErr error
		want    error
	}{
		{"unreachable", secretref.ErrUnreachable, secretref.ErrUnreachable},
		{"token rejected", secretref.ErrNoAccess, secretref.ErrTokenRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, _ := cacheOver(t, func(context.Context) (secretref.Resolver, error) { return nil, tt.openErr })

			_, err := cache.Value(t.Context(), cfRef)

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestCache_RefreshReReadsOnceAndNotAgainWithinFiveMinutes(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{cfRef: "old"}}
	cache, clk := cacheOver(t, opening(fake))
	_, err := cache.Value(t.Context(), cfRef)
	require.NoError(t, err)

	fake.Values[cfRef] = "new"
	fresh, ok := cache.Refresh(t.Context(), "old")
	assert.True(t, ok)
	assert.Equal(t, "new", fresh)
	assert.Len(t, fake.Calls(), 2)

	fake.Values[cfRef] = "newer"
	clk.advance(4 * time.Minute)
	_, ok = cache.Refresh(t.Context(), "new")
	assert.False(t, ok, "a second rejection within 5 minutes does not read again")
	assert.Len(t, fake.Calls(), 2)

	clk.advance(2 * time.Minute)
	fresh, ok = cache.Refresh(t.Context(), "new")
	assert.True(t, ok)
	assert.Equal(t, "newer", fresh)
	assert.Len(t, fake.Calls(), 3)
}

func TestCache_RefreshHandsAConcurrentCallerTheValueAnotherAlreadyRefreshed(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{cfRef: "old"}}
	cache, _ := cacheOver(t, opening(fake))
	_, err := cache.Value(t.Context(), cfRef)
	require.NoError(t, err)
	fake.Values[cfRef] = "new"
	_, ok := cache.Refresh(t.Context(), "old")
	require.True(t, ok)

	fresh, ok := cache.Refresh(t.Context(), "old")

	assert.True(t, ok)
	assert.Equal(t, "new", fresh)
	assert.Len(t, fake.Calls(), 2)
}

func TestCache_RefreshOfAValueThatIsNoReferenceReadsNothing(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{cfRef: "tok"}}
	cache, _ := cacheOver(t, opening(fake))

	_, ok := cache.Refresh(t.Context(), "a-literal-token")

	assert.False(t, ok)
	assert.Empty(t, fake.Calls())
}
