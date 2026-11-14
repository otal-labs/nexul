package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/branding"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestBrandingRepo_SettingsAndLogoLiveInOneRowAndKeepEachOther(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	at := time.Unix(1000, 0).UTC()
	evt := func(id string) eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: id, Topic: branding.TopicChanged, Payload: map[string]string{"id": id}}
	}

	_, found, err := s.Branding.Load(ctx)
	require.NoError(t, err)
	assert.False(t, found, "a fresh instance has no branding")
	_, _, err = s.Branding.Logo(ctx)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	theme := &branding.Theme{Name: "Acme Night", Appearance: branding.Dark, Canvas: "#1b1e28", Text: "#ffffff", Accent: "#7c5cff"}
	require.NoError(t, s.Branding.Save(ctx, branding.Record{ProductName: "Acme Hub", Accent: "#2563eb", Dark: theme, CSS: "a{color:red}", UpdatedBy: "u1", UpdatedAt: at}, evt("e1")))
	require.NoError(t, s.Branding.SetLogo(ctx, "image/png", []byte("png-bytes"), "v1", "u1", at, evt("e2")))

	rec, found, err := s.Branding.Load(ctx)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "Acme Hub", rec.ProductName)
	assert.Equal(t, theme, rec.Dark)
	assert.Equal(t, "a{color:red}", rec.CSS)
	assert.Nil(t, rec.Light)
	assert.Equal(t, &branding.Logo{ContentType: "image/png", Size: 9, Version: "v1"}, rec.Logo)
	ct, data, err := s.Branding.Logo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "image/png", ct)
	assert.Equal(t, []byte("png-bytes"), data)

	require.NoError(t, s.Branding.Save(ctx, branding.Record{ProductName: "Acme", UpdatedBy: "u2", UpdatedAt: at.Add(time.Minute)}))
	rec, _, err = s.Branding.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Acme", rec.ProductName)
	assert.Nil(t, rec.Dark, "saving without a theme removes it")
	assert.NotNil(t, rec.Logo, "saving the settings keeps the logo")

	require.NoError(t, s.Branding.ClearLogo(ctx, "u2", at, evt("e3")))
	rec, _, err = s.Branding.Load(ctx)
	require.NoError(t, err)
	assert.Nil(t, rec.Logo)
	assert.Equal(t, "Acme", rec.ProductName, "removing the logo keeps the settings")
	_, _, err = s.Branding.Logo(ctx)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.Branding.SetLogo(ctx, "image/png", []byte("again"), "v2", "u2", at))
	require.NoError(t, s.Branding.Delete(ctx, evt("e4")))
	_, found, err = s.Branding.Load(ctx)
	require.NoError(t, err)
	assert.False(t, found)
	_, _, err = s.Branding.Logo(ctx)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "clearing everything takes the logo with it")

	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 4)
}
