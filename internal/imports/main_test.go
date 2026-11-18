package imports

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain fails the run when a test leaves the import loop or a download running.
func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
