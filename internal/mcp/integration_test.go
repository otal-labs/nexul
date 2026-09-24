package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/topology"
)

// TestIntegration_StdioSearchDocs is the ws-08 AC integration test: a real
// SQLite store, real in-process bus, outbox relay, and indexing subscriber,
// driven over the stdio transport. Creating a doc publishes doc.created (outbox
// -> relay -> bus -> indexer) while the FTS trigger indexes it; search_docs
// must then return it over stdio.
func TestIntegration_StdioSearchDocs(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewStore(t)
	logger := testutil.DiscardLogger()

	bus := inprocess.New(inprocess.Options{
		Logger:          logger,
		DedupeStore:     store.ProcessedEvents,
		DeadLetterStore: store.DeadLetters,
	})
	defer func() { require.NoError(t, bus.Close()) }()

	docsSvc := docs.NewService(store.Docs, access.NewService(store.Access, testUsers{store.Users}))
	ticketsSvc := tickets.NewService(store.Tickets, store.Statuses, nil)
	topoSvc := topology.NewService(store.Topology)
	deploySvc := deploy.NewService(store.Deploys, store.Stacks, store.Services, testDeployProjects{store.Projects})
	reviewSvc := codereview.NewService(store.CodeReviews)

	owner, _, err := store.Users.UpsertUser(ctx, &auth.User{
		ID:             "owner-user",
		Provider:       auth.ProviderGitHub,
		ProviderUserID: "1",
		Login:          "owner",
		Name:           "Owner",
	})
	require.NoError(t, err)
	require.NoError(t, store.Users.SetCanCreateWorkspace(ctx, owner.ID, true))

	indexer := NewIndexer(logger)
	require.NoError(t, bus.Subscribe(ctx, docs.TopicCreated, indexer.HandleDocEvent))
	require.NoError(t, bus.Subscribe(ctx, docs.TopicUpdated, indexer.HandleDocEvent))
	require.NoError(t, bus.Subscribe(ctx, tickets.TopicCreated, indexer.HandleTicketCreated))

	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Logger: logger, Interval: 10 * time.Millisecond})
	relayCtx, stopRelay := context.WithCancel(ctx)
	defer stopRelay()
	go func() { _ = relay.Run(relayCtx) }()

	doc, err := docsSvc.Create(identity.WithActor(ctx, identity.Actor{ID: owner.ID, CanCreateWorkspace: true}), "project-general", "Storage Spine", "SQLite migrations and FTS5")
	require.NoError(t, err)
	_, err = ticketsSvc.Create(ctx, "project-general", "Fix auth", "login flow broken", "", "")
	require.NoError(t, err)

	accessSvc := access.NewService(store.Access, testUsers{store.Users})
	srv := New(RegistryOptions{
		Docs:       docsSvc,
		Tickets:    ticketsSvc,
		Topology:   topoSvc,
		Deploy:     deploySvc,
		Reviews:    reviewSvc,
		Git:        fakeGitProvider{},
		Access:     accessSvc,
		DeadLetter: store.DeadLetters,
		Publisher:  bus,
		Logger:     logger,
		Actor: func(context.Context) identity.Actor {
			return identity.Actor{ID: owner.ID, CanCreateWorkspace: true}
		},
	})

	client, server := net.Pipe()
	defer func() { require.NoError(t, client.Close()) }()
	serverDone := make(chan struct{})
	go func() {
		_ = srv.ServeStdio(ctx, server, server)
		_ = server.Close()
		close(serverDone)
	}()
	defer func() {
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
			t.Error("stdio server did not exit")
		}
	}()

	enc := json.NewEncoder(client)
	dec := json.NewDecoder(client)

	initResp := stdioCall(t, enc, dec, 1, MethodInitialize, map[string]any{"protocolVersion": "2025-03-26"})
	require.Nil(t, initResp.Error)
	initResult := decodeResult[initializeResult](t, &initResp)
	assert.Equal(t, ProtocolVersion, initResult.ProtocolVersion)

	// search_docs returns the doc the FTS trigger indexed
	searchResp := stdioCall(t, enc, dec, 2, MethodToolsCall, map[string]any{"name": "search_docs", "arguments": map[string]any{"query": "sqlite"}})
	require.Nil(t, searchResp.Error)
	searchResult := decodeResult[toolCallResult](t, &searchResp)
	require.Len(t, searchResult.Content, 1)
	assert.Contains(t, searchResult.Content[0].Text, doc.ID)

	// outbox drained: relay published doc.created + ticket.created
	require.Eventually(t, func() bool {
		pending, err := store.Outbox.Unpublished(ctx, 10)
		return err == nil && len(pending) == 0
	}, 3*time.Second, 20*time.Millisecond)

	// indexer acked the events, so the dead letter store stayed empty and search stays queryable end to end
	searchResp = stdioCall(t, enc, dec, 3, MethodToolsCall, map[string]any{"name": "search_tickets", "arguments": map[string]any{"query": "auth"}})
	require.Nil(t, searchResp.Error)
	searchResult = decodeResult[toolCallResult](t, &searchResp)
	require.Len(t, searchResult.Content, 1)
	assert.Contains(t, searchResult.Content[0].Text, "Fix auth")

	_ = client.Close()
	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stdio server did not stop after client close")
	}
}

func stdioCall(t *testing.T, enc *json.Encoder, dec *json.Decoder, id int, method string, params any) Response {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		require.NoError(t, err)
		raw = b
	}
	req := Request{JSONRPC: "2.0", ID: json.RawMessage(fmt.Sprintf("%d", id)), Method: method, Params: raw}
	require.NoError(t, enc.Encode(req))
	var resp Response
	require.NoError(t, dec.Decode(&resp))
	return resp
}
