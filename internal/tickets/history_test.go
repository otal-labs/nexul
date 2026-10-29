package tickets

import (
	"context"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// The use-case tests here keep no history; the real store's tests in internal/platform/storage hold what is written.
func (*fakeRepo) ListDescriptionVersions(context.Context, string) ([]VersionSummary, error) {
	return nil, nil
}

func (*fakeRepo) GetDescriptionVersion(context.Context, string, int) (*DescriptionVersion, error) {
	return nil, apperrs.ErrNotFound
}

func (*fakeRepo) ListChanges(context.Context, string) ([]Change, error) { return nil, nil }

func (*fakeRepo) AppendSittingVersion(context.Context, string, []string, time.Time) error {
	return nil
}
