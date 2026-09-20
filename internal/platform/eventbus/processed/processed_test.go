package processed

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
)

// TestScope_NamespacesKeysPerConsumer proves Scope gives each consumer its own
// dedupe namespace: two scopes over the same store never see each other's
// records, and the raw key stays untouched.
func TestScope_NamespacesKeysPerConsumer(t *testing.T) {
	ctx := context.Background()
	base := testutil.NewStore(t).ProcessedEvents

	a := Scope(base, "consumer-a")
	b := Scope(base, "consumer-b")

	require.NoError(t, a.Record(ctx, "e1"))
	require.NoError(t, b.Record(ctx, "e1"))

	seenA, err := a.Seen(ctx, "e1")
	require.NoError(t, err)
	assert.True(t, seenA, "consumer-a must see its own record")
	seenB, err := b.Seen(ctx, "e1")
	require.NoError(t, err)
	assert.True(t, seenB, "consumer-b must see its own record")

	seenBase, err := base.Seen(ctx, "e1")
	require.NoError(t, err)
	assert.False(t, seenBase, "the raw key must stay untouched by scoped records")

	seenOther, err := a.Seen(ctx, "e2")
	require.NoError(t, err)
	assert.False(t, seenOther)
}
