package dns

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func testKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func TestService_VerifyCredentials(t *testing.T) {
	t.Run("verify failure with bad token is fatal", func(t *testing.T) {
		p := newFakeProvider()
		p.verifyErr = apperrs.ErrUnauthorized
		s := newTestService(newFakeRepo(), p, nil)
		err := s.VerifyCredentials(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("verify failure with transient error stays retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.verifyErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		err := s.VerifyCredentials(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
		assert.False(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("verify ok", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		require.NoError(t, s.VerifyCredentials(context.Background()))
	})
}

func TestService_ListZones(t *testing.T) {
	t.Run("returns zones", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		zones, err := s.ListZones(context.Background())
		require.NoError(t, err)
		require.Len(t, zones, 1)
		assert.Equal(t, "example.com", zones[0].Name)
	})
	t.Run("missing zone is fatal", func(t *testing.T) {
		p := newFakeProvider()
		p.zonesErr = apperrs.ErrNotFound
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.ListZones(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func TestService_ListRecords(t *testing.T) {
	t.Run("empty zone id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.ListRecords(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("provider not found is fatal", func(t *testing.T) {
		p := newFakeProvider()
		p.listErr = apperrs.ErrNotFound
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.ListRecords(context.Background(), "z1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("provider transient error stays retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.listErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.ListRecords(context.Background(), "z1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("lists records", func(t *testing.T) {
		p := newFakeProvider()
		_, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
		require.NoError(t, err)
		s := newTestService(newFakeRepo(), p, nil)
		records, err := s.ListRecords(context.Background(), "z1")
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, "api", records[0].Name)
	})
}

func TestService_CreateRecord(t *testing.T) {
	t.Run("validation errors", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		tests := []struct {
			name string
			in   RecordInput
		}{
			{"unsupported type", RecordInput{Type: "SRV", Name: "x", Content: "y"}},
			{"empty name", RecordInput{Type: RecordA, Name: "", Content: "1.2.3.4"}},
			{"empty content", RecordInput{Type: RecordA, Name: "x", Content: ""}},
			{"negative ttl", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4", TTL: -2}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := s.CreateRecord(context.Background(), "z1", tt.in)
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
			})
		}
	})
	t.Run("empty zone id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.CreateRecord(context.Background(), "", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("provider transient error stays retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.createErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("bad token is fatal", func(t *testing.T) {
		p := newFakeProvider()
		p.createErr = apperrs.ErrUnauthorized
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("creates record and publishes outbox event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeProvider(), nil)
		rec, err := s.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 300})
		require.NoError(t, err)
		assert.Equal(t, "api", rec.Name)
		assert.Equal(t, TopicRecordChanged, repo.topics()[0])

		var evt RecordChangedEvent
		require.NoError(t, json.Unmarshal(repo.outbox[0].Payload.(json.RawMessage), &evt))
		assert.Equal(t, "created", evt.Action)
		assert.Equal(t, rec.ID, evt.RecordID)
		assert.Empty(t, evt.Service)
	})
}

func TestService_UpdateRecord(t *testing.T) {
	t.Run("missing ids are invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.UpdateRecord(context.Background(), "z1", "", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing record is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.UpdateRecord(context.Background(), "z1", "nope", RecordInput{Type: RecordA, Name: "x", Content: "1.2.3.4"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("updates record and publishes event", func(t *testing.T) {
		repo := newFakeRepo()
		p := newFakeProvider()
		created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
		require.NoError(t, err)
		s := newTestService(repo, p, nil)
		got, err := s.UpdateRecord(context.Background(), "z1", created.ID, RecordInput{Type: RecordA, Name: "api", Content: "5.6.7.8", TTL: 60})
		require.NoError(t, err)
		assert.Equal(t, "5.6.7.8", got.Content)
		assert.Equal(t, "updated", outboxAction(t, repo, 0))
	})
}

func TestService_DeleteRecord(t *testing.T) {
	t.Run("missing ids are invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		err := s.DeleteRecord(context.Background(), "", "r1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("idempotent delete of absent record publishes event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeProvider(), nil)
		require.NoError(t, s.DeleteRecord(context.Background(), "z1", "ghost"))
		assert.Equal(t, "deleted", outboxAction(t, repo, 0))
	})
	t.Run("provider failure stays retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.deleteErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		err := s.DeleteRecord(context.Background(), "z1", "r1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestService_CheckPropagation(t *testing.T) {
	t.Run("missing zone or record is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		err := s.CheckPropagation(context.Background(), "", Record{ID: "r1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("not propagated yet is retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.propagateErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		err := s.CheckPropagation(context.Background(), "z1", Record{ID: "r1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("propagated", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		require.NoError(t, s.CheckPropagation(context.Background(), "z1", Record{ID: "r1", Name: "api.example.com", Content: "1.2.3.4", Type: RecordA}))
	})
}

func TestService_CreateInstanceRecord(t *testing.T) {
	t.Run("missing zone is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.CreateInstanceRecord(context.Background(), "", "", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing target is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("instance url not set yet", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		s.settings = &fakeSettings{err: apperrs.ErrNotFound}
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid instance url", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		s.settings = &fakeSettings{instanceURL: "not-a-url"}
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("host not under zone surfaces mismatch", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "other.com", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		assert.Contains(t, err.Error(), "not under zone")
	})
	t.Run("creates record derived from instance url host", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeProvider(), nil)
		rec, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "10.0.0.1")
		require.NoError(t, err)
		assert.Equal(t, "deploy", rec.Name, "record name derives from deploy.example.com under example.com")
		assert.Equal(t, "10.0.0.1", rec.Content)
		assert.Equal(t, "created", outboxAction(t, repo, 0))
	})
	t.Run("apex instance url", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		s.settings = &fakeSettings{instanceURL: "https://example.com"}
		rec, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "10.0.0.1")
		require.NoError(t, err)
		assert.Equal(t, "@", rec.Name)
	})
}

func TestService_SetServiceHostname(t *testing.T) {
	t.Run("validation errors", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.SetServiceHostname(context.Background(), ServiceHostnameInput{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("hostname not under zone surfaces mismatch", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.SetServiceHostname(context.Background(), validHostnameInput())
		require.NoError(t, err)
		in := validHostnameInput()
		in.Hostname = "api.other.com"
		_, err = s.SetServiceHostname(context.Background(), in)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("provider failure stays retryable", func(t *testing.T) {
		p := newFakeProvider()
		p.createErr = apperrs.Retryable(errBoom)
		s := newTestService(newFakeRepo(), p, nil)
		_, err := s.SetServiceHostname(context.Background(), validHostnameInput())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("creates record, stores association, publishes outbox event in one tx", func(t *testing.T) {
		repo := newFakeRepo()
		p := newFakeProvider()
		s := newTestService(repo, p, nil)
		sh, err := s.SetServiceHostname(context.Background(), validHostnameInput())
		require.NoError(t, err)
		assert.Equal(t, "api.example.com", sh.Hostname)
		assert.Equal(t, "r1", sh.RecordID)

		stored, err := repo.GetServiceHostname(context.Background(), "api")
		require.NoError(t, err)
		assert.Equal(t, sh.RecordID, stored.RecordID)

		var evt RecordChangedEvent
		require.NoError(t, json.Unmarshal(repo.outbox[0].Payload.(json.RawMessage), &evt))
		assert.Equal(t, "created", evt.Action)
		assert.Equal(t, "api", evt.Service)
	})
}

func TestService_GetServiceHostname(t *testing.T) {
	t.Run("empty service is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.GetServiceHostname(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing association is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		_, err := s.GetServiceHostname(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestService_ListServiceHostnames(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakeProvider(), nil)
	_, err := s.SetServiceHostname(context.Background(), validHostnameInput())
	require.NoError(t, err)
	list, err := s.ListServiceHostnames(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "api", list[0].Service)
}

func TestService_RemoveServiceHostname(t *testing.T) {
	t.Run("empty service is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		err := s.RemoveServiceHostname(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing association is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		err := s.RemoveServiceHostname(context.Background(), "ghost")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("deletes provider record and association", func(t *testing.T) {
		repo := newFakeRepo()
		p := newFakeProvider()
		s := newTestService(repo, p, nil)
		sh, err := s.SetServiceHostname(context.Background(), validHostnameInput())
		require.NoError(t, err)
		require.NoError(t, s.RemoveServiceHostname(context.Background(), "api"))

		_, err = repo.GetServiceHostname(context.Background(), "api")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound), "association removed")
		records, err := p.ListRecords(context.Background(), "z1")
		require.NoError(t, err)
		assert.Len(t, records, 0, "provider record removed")
		assert.Equal(t, "deleted", outboxAction(t, repo, 1))
		assert.Equal(t, "api", sh.Service)
	})
}

func TestService_ResolveToken(t *testing.T) {
	t.Run("token provider result is passed to the provider constructor", func(t *testing.T) {
		var got string
		cfg := Config{
			Repo:   newFakeRepo(),
			Tokens: &fakeTokenProvider{token: "cf-live-token"},
			NewProvider: func(_ context.Context, token string) (DNSProvider, error) {
				got = token
				return newFakeProvider(), nil
			},
			EncryptionKey: testKey(), Now: time.Now,
		}
		s := NewService(cfg)
		_, err := s.ListZones(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "cf-live-token", got)
	})
	t.Run("no token source wired is fatal invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), nil, nil)
		s.tokens = nil
		_, err := s.ListZones(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("not connected is fatal retryable", func(t *testing.T) {
		s := newTestService(newFakeRepo(), nil, &fakeTokenProvider{err: apperrs.ErrNotFound})
		_, err := s.ListZones(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("token provider failure is fatal retryable", func(t *testing.T) {
		s := newTestService(newFakeRepo(), nil, &fakeTokenProvider{err: errBoom})
		_, err := s.ListZones(context.Background())
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestService_UnclassifiedProviderErrorPassesThrough(t *testing.T) {
	p := newFakeProvider()
	p.zonesErr = errBoom // not a platform sentinel — passes through untouched, not fatal/retryable
	s := newTestService(newFakeRepo(), p, nil)
	_, err := s.ListZones(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, errBoom)
	assert.False(t, errors.Is(err, apperrs.ErrFatal))
}

func TestService_InstanceURL_SettingsErrors(t *testing.T) {
	t.Run("settings reader missing", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		s.settings = nil
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("settings read failure surfaces", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeProvider(), nil)
		s.settings = &fakeSettings{err: errBoom}
		_, err := s.CreateInstanceRecord(context.Background(), "z1", "example.com", RecordA, "1.2.3.4")
		require.Error(t, err)
		assert.ErrorIs(t, err, errBoom)
	})
}

func TestNewService_DefaultsClock(t *testing.T) {
	s := NewService(Config{Repo: newFakeRepo(), Provider: newFakeProvider(), EncryptionKey: testKey()})
	require.NotNil(t, s.now)
	_, err := s.ListZones(context.Background())
	require.NoError(t, err)
}

func validHostnameInput() ServiceHostnameInput {
	return ServiceHostnameInput{
		Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com",
		Type: RecordA, Target: "1.2.3.4",
	}
}

func outboxAction(t *testing.T, repo *fakeRepo, i int) string {
	t.Helper()
	require.Greater(t, len(repo.outbox), i)
	raw, ok := repo.outbox[i].Payload.(json.RawMessage)
	require.True(t, ok)
	var evt RecordChangedEvent
	require.NoError(t, json.Unmarshal(raw, &evt))
	return evt.Action
}
