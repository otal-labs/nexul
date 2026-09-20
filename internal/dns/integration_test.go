package dns_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// openStore returns a migrated real-SQLite store on a temp file.
func openStore(t *testing.T) *storage.Store {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
}

type recvBus struct {
	ch chan string
}

func (b *recvBus) Publish(_ context.Context, topic string, _ any) error {
	b.ch <- topic
	return nil
}

func (b *recvBus) PublishWithID(_ context.Context, _ string, topic string, _ any) error {
	return b.Publish(context.TODO(), topic, nil)
}

// fakeProvider is the minimal provider needed for a hostname association.
type fakeProvider struct{}

func (fakeProvider) Verify(context.Context) error { return nil }
func (fakeProvider) ListZones(context.Context) ([]dns.Zone, error) {
	return []dns.Zone{{ID: "z1", Name: "example.com"}}, nil
}
func (fakeProvider) ListRecords(context.Context, string) ([]dns.Record, error) { return nil, nil }
func (fakeProvider) CreateRecord(_ context.Context, zoneID string, in dns.RecordInput) (*dns.Record, error) {
	return &dns.Record{ID: "r1", ZoneID: zoneID, Type: in.Type, Name: in.Name, Content: in.Content, TTL: in.TTL}, nil
}
func (fakeProvider) UpdateRecord(_ context.Context, zoneID, recordID string, in dns.RecordInput) (*dns.Record, error) {
	return &dns.Record{ID: recordID, ZoneID: zoneID, Type: in.Type, Name: in.Name, Content: in.Content, TTL: in.TTL}, nil
}
func (fakeProvider) DeleteRecord(context.Context, string, string) error { return nil }
func (fakeProvider) CheckPropagation(context.Context, string, dns.Record) error {
	return nil
}

// TestIntegration_RecordChanged_ReachesOutboxAndRelay verifies that a dns
// mutation writes dns.record_changed into the transactional outbox and the
// relay then publishes it (outbox-backed), over real SQLite.
func TestIntegration_RecordChanged_ReachesOutboxAndRelay(t *testing.T) {
	store := openStore(t)
	svc := dns.NewService(dns.Config{
		Repo:          store.DNS,
		Provider:      fakeProvider{},
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		Settings:      instanceSettings{},
		Now:           time.Now,
	})

	_, err := svc.SetServiceHostname(context.Background(), dns.ServiceHostnameInput{
		Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com",
		Type: dns.RecordA, Target: "1.2.3.4",
	})
	require.NoError(t, err)

	rows, err := store.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, rows, 1, "exactly one outbox row for one mutation")
	assert.Equal(t, dns.TopicRecordChanged, rows[0].Topic)

	var evt dns.RecordChangedEvent
	require.NoError(t, json.Unmarshal(rows[0].Payload, &evt))
	assert.Equal(t, "created", evt.Action)
	assert.Equal(t, "api", evt.Service)

	ctx, cancel := context.WithCancel(context.Background())
	bus := &recvBus{ch: make(chan string, 4)}
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Interval: 5 * time.Millisecond})
	go func() {
		_ = relay.Run(ctx) // only returns after ctx cancellation, always nil
	}()
	defer cancel()

	select {
	case topic := <-bus.ch:
		assert.Equal(t, dns.TopicRecordChanged, topic)
	case <-time.After(2 * time.Second):
		t.Fatal("relay never published the outbox row")
	}
}

type instanceSettings struct{}

func (instanceSettings) GetInstanceURL(context.Context) (string, error) {
	return "https://deploy.example.com", nil
}
