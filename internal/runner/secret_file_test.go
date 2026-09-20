package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSecretFile(t *testing.T) {
	t.Run("waits for the server to publish the file, then trims it", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "runner-secret")
		go func() {
			time.Sleep(50 * time.Millisecond)
			_ = os.WriteFile(path, []byte("s3cr3t\n"), 0o600)
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		got, err := ReadSecretFile(ctx, path)
		require.NoError(t, err)
		assert.Equal(t, "s3cr3t", got)
	})

	t.Run("gives up when the context ends first", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := ReadSecretFile(ctx, filepath.Join(t.TempDir(), "never"))
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
