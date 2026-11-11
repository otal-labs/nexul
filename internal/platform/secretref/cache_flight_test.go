package secretref_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

const slowRef = "op://Acme/slow/field"

// gatedResolver holds every read of slowRef until release is closed; other references answer at once.
type gatedResolver struct {
	secretreftest.Fake
	entered chan struct{}
	release chan struct{}
}

func newGated() *gatedResolver {
	return &gatedResolver{
		Fake:    secretreftest.Fake{Values: map[string]string{slowRef: "slow-value", cfRef: "tok-1"}},
		entered: make(chan struct{}, 8),
		release: make(chan struct{}),
	}
}

func (g *gatedResolver) ResolveAll(ctx context.Context, refs []string) ([]secretref.Result, error) {
	if len(refs) == 1 && refs[0] == slowRef {
		g.entered <- struct{}{}
		<-g.release
	}
	return g.Fake.ResolveAll(ctx, refs)
}

func TestCache_ASlowReadOfOneReferenceDoesNotBlockAnother(t *testing.T) {
	gated := newGated()
	cache, _ := cacheOver(t, opening(gated))
	slow := make(chan string, 1)
	go func() {
		v, _ := cache.Value(context.Background(), slowRef)
		slow <- v
	}()
	<-gated.entered

	got, err := cache.Value(t.Context(), cfRef)

	require.NoError(t, err)
	assert.Equal(t, "tok-1", got, "answered while the other reference's read is still running")
	close(gated.release)
	assert.Equal(t, "slow-value", <-slow)
}

func TestCache_ConcurrentLookupsOfOneReferenceShareOneRead(t *testing.T) {
	gated := newGated()
	cache, _ := cacheOver(t, opening(gated))
	var wg sync.WaitGroup
	got := make([]string, 3)
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got[i], _ = cache.Value(context.Background(), slowRef)
		}()
	}
	<-gated.entered
	time.Sleep(20 * time.Millisecond)
	close(gated.release)
	wg.Wait()

	assert.Equal(t, []string{"slow-value", "slow-value", "slow-value"}, got)
	assert.Len(t, gated.Calls(), 1)
}

func TestCache_WarmReadsEveryReferenceOfAManagerInOneCallAndSeedsTheCache(t *testing.T) {
	const gone = "op://Acme/gone/field"
	fake := &secretreftest.Fake{
		Values: map[string]string{cfRef: "tok-1", slowRef: "tok-2"},
		Errs:   map[string]error{gone: secretref.ErrNotFound},
	}
	cache, _ := cacheOver(t, opening(fake))

	cache.Warm(t.Context(), []string{cfRef, slowRef, cfRef, "plain-token", gone})

	require.Len(t, fake.Calls(), 1)
	assert.ElementsMatch(t, []string{cfRef, slowRef, gone}, fake.Calls()[0])
	for ref, want := range map[string]string{cfRef: "tok-1", slowRef: "tok-2"} {
		got, err := cache.Value(t.Context(), ref)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := cache.Value(t.Context(), gone)
	require.ErrorIs(t, err, secretref.ErrNotFound)
	assert.Len(t, fake.Calls(), 1, "no lookup after the warm read went back to the manager")
}

// hangingResolver answers only when its context ends, the way a stalled network call does.
type hangingResolver struct{ secretreftest.Fake }

func (h *hangingResolver) ResolveAll(ctx context.Context, _ []string) ([]secretref.Result, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCache_AReadThatTimesOutIsUnreachable(t *testing.T) {
	cache, _ := cacheOver(t, opening(&hangingResolver{}))
	cache.SetReadTimeout(10 * time.Millisecond)

	_, err := cache.Read(t.Context(), cfRef)

	require.ErrorIs(t, err, secretref.ErrUnreachable)
	assert.NotErrorIs(t, err, context.DeadlineExceeded)
}
