package integrations

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeData is the shared in-memory backing store for the per-interface fakes
// below (the testing standard: prefer fakes over mocks). Each store type wraps
// the same data with the interface's exact method set, so no name collisions.
type fakeData struct {
	mu         sync.Mutex
	installs   map[string]*Install
	tokens     map[string]*IntegrationToken
	subs       map[string]Subscription
	deliveries map[string]*Delivery
	schemas    map[string]SchemaEntry
	audit      []AuditEntry
}

func newFakeData() *fakeData {
	return &fakeData{
		installs:   map[string]*Install{},
		tokens:     map[string]*IntegrationToken{},
		subs:       map[string]Subscription{},
		deliveries: map[string]*Delivery{},
		schemas:    map[string]SchemaEntry{},
	}
}

type fakeInstallStore struct{ d *fakeData }

func (f *fakeInstallStore) Create(ctx context.Context, install *Install) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	f.d.installs[install.ID] = install
	return nil
}

func (f *fakeInstallStore) GetByID(ctx context.Context, id string) (*Install, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	install, ok := f.d.installs[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *install
	return &cp, nil
}

func (f *fakeInstallStore) List(ctx context.Context) ([]*Install, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	out := make([]*Install, 0, len(f.d.installs))
	for _, install := range f.d.installs {
		cp := *install
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeInstallStore) Revoke(ctx context.Context, id string) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	install, ok := f.d.installs[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	now := time.Now().UTC()
	install.RevokedAt = &now
	return nil
}

type fakeTokenStore struct{ d *fakeData }

func (f *fakeTokenStore) Create(ctx context.Context, token *IntegrationToken) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	f.d.tokens[token.ID] = token
	return nil
}

func (f *fakeTokenStore) GetByHash(ctx context.Context, hash string) (*IntegrationToken, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	for _, token := range f.d.tokens {
		if token.TokenHash == hash {
			cp := *token
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeTokenStore) ListByInstall(ctx context.Context, installID string) ([]*IntegrationToken, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []*IntegrationToken
	for _, token := range f.d.tokens {
		if token.InstallID == installID {
			cp := *token
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeTokenStore) RevokeByInstall(ctx context.Context, installID string) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	for _, token := range f.d.tokens {
		if token.InstallID == installID {
			now := time.Now().UTC()
			token.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeTokenStore) TouchLastUsed(ctx context.Context, id string) error {
	return nil
}

type fakeSubStore struct{ d *fakeData }

func (f *fakeSubStore) Add(ctx context.Context, sub Subscription) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	key := sub.InstallID + "|" + sub.Topic
	if _, ok := f.d.subs[key]; ok {
		return apperrs.ErrConflict
	}
	f.d.subs[key] = sub
	return nil
}

func (f *fakeSubStore) Remove(ctx context.Context, installID, topic string) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	key := installID + "|" + topic
	if _, ok := f.d.subs[key]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.d.subs, key)
	return nil
}

func (f *fakeSubStore) ListByInstall(ctx context.Context, installID string) ([]Subscription, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []Subscription
	for _, sub := range f.d.subs {
		if sub.InstallID == installID {
			out = append(out, sub)
		}
	}
	return out, nil
}

func (f *fakeSubStore) ListByTopic(ctx context.Context, topic string) ([]Subscription, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []Subscription
	for _, sub := range f.d.subs {
		if sub.Topic == topic {
			out = append(out, sub)
		}
	}
	return out, nil
}

type fakeDeliveryStore struct{ d *fakeData }

func (f *fakeDeliveryStore) Create(ctx context.Context, d *Delivery) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	key := d.InstallID + "|" + d.EventID + "|" + d.Topic
	for _, existing := range f.d.deliveries {
		if existing.InstallID+"|"+existing.EventID+"|"+existing.Topic == key {
			return nil // idempotent, mirrors ON CONFLICT DO NOTHING
		}
	}
	f.d.deliveries[d.ID] = d
	return nil
}

func (f *fakeDeliveryStore) Due(ctx context.Context, limit int, now int64) ([]*Delivery, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []*Delivery
	for _, d := range f.d.deliveries {
		if (d.Status == DeliveryPending || d.Status == DeliveryFailed) && d.NextAttemptAt.Unix() <= now {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeDeliveryStore) MarkDelivered(ctx context.Context, id string, at int64) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	d, ok := f.d.deliveries[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Status = DeliveryDelivered
	t := time.Unix(at, 0).UTC()
	d.DeliveredAt = &t
	return nil
}

func (f *fakeDeliveryStore) MarkFailed(ctx context.Context, id string, attempts int, nextAttemptAt int64) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	d, ok := f.d.deliveries[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Status = DeliveryFailed
	d.Attempts = attempts
	d.NextAttemptAt = time.Unix(nextAttemptAt, 0).UTC()
	return nil
}

func (f *fakeDeliveryStore) MarkDead(ctx context.Context, id string) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	d, ok := f.d.deliveries[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.Status = DeliveryDead
	d.Attempts++
	return nil
}

func (f *fakeDeliveryStore) ListByInstall(ctx context.Context, installID string) ([]*Delivery, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []*Delivery
	for _, d := range f.d.deliveries {
		if d.InstallID == installID {
			out = append(out, d)
		}
	}
	return out, nil
}

type fakeSchemaStore struct{ d *fakeData }

func (f *fakeSchemaStore) Publish(ctx context.Context, entry SchemaEntry) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	key := entry.Topic + "|" + itoa(entry.Version)
	if _, ok := f.d.schemas[key]; ok {
		return nil
	}
	f.d.schemas[key] = entry
	return nil
}

func (f *fakeSchemaStore) Catalog(ctx context.Context) ([]SchemaEntry, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	var out []SchemaEntry
	for _, entry := range f.d.schemas {
		out = append(out, entry)
	}
	return out, nil
}

type fakeAuditStore struct{ d *fakeData }

func (f *fakeAuditStore) Append(ctx context.Context, entry AuditEntry) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	f.d.audit = append(f.d.audit, entry)
	return nil
}

func (f *fakeAuditStore) List(ctx context.Context, limit int) ([]AuditEntry, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	out := make([]AuditEntry, 0, len(f.d.audit))
	out = append(out, f.d.audit...)
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

type fakeOwner struct{ ok bool }

func (o fakeOwner) CanCreateWorkspace(ctx context.Context, userID string) (bool, error) {
	return o.ok, nil
}

func newTestServiceWithOwner(f *fakeData, allowCreate bool) *Service {
	return NewService(Config{
		Installs:   &fakeInstallStore{d: f},
		Tokens:     &fakeTokenStore{d: f},
		Subs:       &fakeSubStore{d: f},
		Deliveries: &fakeDeliveryStore{d: f},
		Schemas:    &fakeSchemaStore{d: f},
		Audit:      &fakeAuditStore{d: f},
		Owner:      fakeOwner{ok: allowCreate},
		Now:        func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	})
}

func TestInstall(t *testing.T) {
	t.Run("mints scoped token with expanded scopes", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		raw, secret, install, err := svc.Install(context.Background(), "user-1", "discord", TrustCommunity, "https://example.com/hook", []Scope{ScopeTicketsWrite})
		require.NoError(t, err)
		require.NotEmpty(t, raw)
		require.NotEmpty(t, secret)
		assert.True(t, len(raw) > 6)
		assert.Equal(t, "discord", install.Name)
		assert.Contains(t, install.Scopes, ScopeTicketsRead)
		assert.NotContains(t, install.Scopes, ScopeProjectsRead)
		assert.NotEqual(t, secret, "")
	})

	t.Run("raw token authenticates and install scopes are enforced", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		raw, _, _, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, "", []Scope{ScopeDocsWrite})
		require.NoError(t, err)
		install, _, err := svc.AuthenticateToken(context.Background(), raw)
		require.NoError(t, err)
		assert.Contains(t, install.Scopes, ScopeDocsRead)
	})

	t.Run("error paths", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		tests := []struct {
			name    string
			nameArg string
			tier    TrustTier
			url     string
			scopes  []Scope
		}{
			{"empty name", "", TrustVerified, "", []Scope{ScopeDocsRead}},
			{"bad tier", "x", "garbage", "", []Scope{ScopeDocsRead}},
			{"bad webhook url", "x", TrustVerified, "ftp://example.com", []Scope{ScopeDocsRead}},
			{"url with credentials", "x", TrustVerified, "https://user:pass@example.com", []Scope{ScopeDocsRead}},
			{"no scopes", "x", TrustVerified, "", nil},
			{"invalid scope", "x", TrustVerified, "", []Scope{"bogus:read"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, _, err := svc.Install(context.Background(), "user-1", tt.nameArg, tt.tier, tt.url, tt.scopes)
				require.Error(t, err)
				assert.True(t, errors.Is(err, apperrs.ErrInvalid), "got %v", err)
			})
		}
	})

	t.Run("non-owner denied", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, false)
		_, _, _, err := svc.Install(context.Background(), "user-1", "x", TrustVerified, "", []Scope{ScopeDocsRead})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestAuthenticateToken(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	raw, _, install, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, "", []Scope{ScopeEventsRead})
	require.NoError(t, err)

	t.Run("valid token resolves install", func(t *testing.T) {
		got, _, err := svc.AuthenticateToken(context.Background(), raw)
		require.NoError(t, err)
		assert.Equal(t, install.ID, got.ID)
	})

	t.Run("wrong prefix rejected", func(t *testing.T) {
		_, _, err := svc.AuthenticateToken(context.Background(), "not_"+raw)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})

	t.Run("unknown token rejected", func(t *testing.T) {
		_, _, err := svc.AuthenticateToken(context.Background(), tokenPrefix+"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})

	t.Run("revoked install rejected immediately", func(t *testing.T) {
		raw2, _, install2, err := svc.Install(context.Background(), "user-1", "y", TrustCommunity, "", []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.RevokeInstall(context.Background(), "user-1", install2.ID))
		_, _, err = svc.AuthenticateToken(context.Background(), raw2)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
}

func TestSubscribeUnsubscribe(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	_, _, install, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, "https://example.com", []Scope{ScopeEventsRead})
	require.NoError(t, err)

	t.Run("subscribe then list", func(t *testing.T) {
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))
		subs, err := svc.ListSubscriptions(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		require.Len(t, subs, 1)
		assert.Equal(t, "ticket.created", subs[0].Topic)
	})

	t.Run("duplicate subscribe conflicts", func(t *testing.T) {
		err := svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})

	t.Run("subscribe unknown install", func(t *testing.T) {
		err := svc.Subscribe(context.Background(), "user-1", "nope", "ticket.created")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})

	t.Run("empty topic invalid", func(t *testing.T) {
		err := svc.Subscribe(context.Background(), "user-1", install.ID, "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("unsubscribe", func(t *testing.T) {
		require.NoError(t, svc.Unsubscribe(context.Background(), "user-1", install.ID, "ticket.created"))
		subs, err := svc.ListSubscriptions(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Empty(t, subs)
	})

	t.Run("unsubscribe missing not found", func(t *testing.T) {
		err := svc.Unsubscribe(context.Background(), "user-1", install.ID, "ticket.created")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestPublishSchemaAndCatalog(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)

	require.NoError(t, svc.cfg.Schemas.Publish(context.Background(), SchemaEntry{Topic: "ticket.created", Version: 1, Schema: `{"type":"object"}`, CreatedAt: time.Now().UTC()}))
	require.NoError(t, svc.cfg.Schemas.Publish(context.Background(), SchemaEntry{Topic: "ticket.created", Version: 2, Schema: `{"type":"object","x":1}`, CreatedAt: time.Now().UTC()}))

	entries, err := svc.Catalog(context.Background())
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestAudit(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	require.NoError(t, svc.cfg.Audit.Append(context.Background(), AuditEntry{
		ID: "a1", ActorType: "integration", ActorID: "install-1", TokenID: "token-1", Action: "GET /api/tickets",
		CreatedAt: time.Now().UTC(),
	}))
	entries, err := svc.ListAudit(context.Background(), "user-1", 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "token-1", entries[0].TokenID)

	t.Run("non-owner denied", func(t *testing.T) {
		f2 := newFakeData()
		svc2 := newTestServiceWithOwner(f2, false)
		_, err := svc2.ListAudit(context.Background(), "user-1", 10)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestListInstallsGetInstall(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	_, _, install, err := svc.Install(context.Background(), "user-1", "discord", TrustCommunity, "", []Scope{ScopeEventsRead})
	require.NoError(t, err)

	installs, err := svc.ListInstalls(context.Background(), "user-1")
	require.NoError(t, err)
	require.Len(t, installs, 1)

	got, err := svc.GetInstall(context.Background(), "user-1", install.ID)
	require.NoError(t, err)
	assert.Equal(t, install.ID, got.ID)

	t.Run("get unknown install not found", func(t *testing.T) {
		_, err := svc.GetInstall(context.Background(), "user-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}
