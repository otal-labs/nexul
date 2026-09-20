package gitprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestListPRs(t *testing.T) {
	t.Run("validates empty owner", func(t *testing.T) {
		_, err := ListPRs(context.Background(), &fakeProvider{}, "", "app", PROpts{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("validates empty repo", func(t *testing.T) {
		_, err := ListPRs(context.Background(), &fakeProvider{}, "acme", "", PROpts{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("returns provider result", func(t *testing.T) {
		want := []*PR{{Number: 1}, {Number: 2}}
		got, err := ListPRs(context.Background(), &fakeProvider{prs: want}, "acme", "app", PROpts{State: "open", Limit: 5})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("propagates provider error", func(t *testing.T) {
		boom := errors.New("api down")
		_, err := ListPRs(context.Background(), &fakeProvider{err: boom}, "acme", "app", PROpts{})
		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
	})
}

func TestGetPR(t *testing.T) {
	t.Run("validates empty owner", func(t *testing.T) {
		_, err := GetPR(context.Background(), &fakeProvider{}, "", "app", 1)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("validates non-positive number", func(t *testing.T) {
		_, err := GetPR(context.Background(), &fakeProvider{}, "acme", "app", 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("returns provider result", func(t *testing.T) {
		want := &PR{Number: 7, Title: "Fix login"}
		got, err := GetPR(context.Background(), &fakeProvider{pr: want}, "acme", "app", 7)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func TestGetRepo(t *testing.T) {
	t.Run("validates empty owner", func(t *testing.T) {
		_, err := GetRepo(context.Background(), &fakeProvider{}, "", "app")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("returns provider result", func(t *testing.T) {
		want := &Repo{Name: "app"}
		got, err := GetRepo(context.Background(), &fakeProvider{repo: want}, "acme", "app")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
