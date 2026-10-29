package mentions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeNamedSource struct {
	named []Named
	err   error
}

func (f fakeNamedSource) Find(context.Context, string, []string) ([]Named, error) {
	return f.named, f.err
}

func TestTitleTier(t *testing.T) {
	tests := []struct {
		name, title, query string
		want               int
	}{
		{"the title is the query", "Deploy Logs", "deploy logs", 3},
		{"each word whole", "Fix the deploy logs page", "logs deploy", 2},
		{"a word only as a prefix", "Deployment logs", "deploy logs", 1},
		{"a word inside another is no match", "Redeploy", "deploy", -1},
		{"a word missing", "Deploy logs", "deploy errors", -1},
		{"punctuation splits words", "web/app#12", "app 12", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, titleTier(tt.title, searchWords(tt.query)))
		})
	}
}

func TestCutRanked_OrdersByTierThenRecency_AndCutsEachSection(t *testing.T) {
	now := time.Now()
	ranked := []rankedHit{
		{tier: 1, hit: Hit{ID: "prefix-new", UpdatedAt: now}},
		{tier: 2, hit: Hit{ID: "whole-old", UpdatedAt: now.Add(-time.Hour)}},
		{tier: 2, hit: Hit{ID: "whole-new", UpdatedAt: now}},
		{tier: -1, hit: Hit{ID: "body-old", InBody: true, UpdatedAt: now.Add(-time.Hour)}},
		{tier: -1, hit: Hit{ID: "body-new", InBody: true, UpdatedAt: now}},
	}
	hits, more := cutRanked(ranked, 2)
	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
	}
	assert.Equal(t, []string{"whole-new", "whole-old", "body-new", "body-old"}, ids)
	assert.True(t, more, "a third title hit was cut")

	_, more = cutRanked(ranked[1:], 2)
	assert.False(t, more)
}

func TestFind_Errors(t *testing.T) {
	tests := []struct {
		name                   string
		ctx                    context.Context
		query, workspace, kind string
		want                   error
	}{
		{"no query", actorCtx("u-1"), " ;; ", "ws-1", "", apperrs.ErrInvalid},
		{"no workspace", actorCtx("u-1"), "deploy", " ", "", apperrs.ErrInvalid},
		{"an unknown kind", actorCtx("u-1"), "deploy", "ws-1", "person", apperrs.ErrInvalid},
		{"no actor", context.Background(), "deploy", "ws-1", "", apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, nil, nil, nil, nil)
			_, err := svc.Find(tt.ctx, tt.query, tt.workspace, tt.kind, 5)
			require.ErrorIs(t, err, tt.want)
		})
	}
	t.Run("unwired sources", func(t *testing.T) {
		_, err := New(Config{}).Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
		require.Error(t, err)
	})
}

func TestFind_TicketSearchFailure_FailsTheAnswer(t *testing.T) {
	svc := newTestService(t, &fakeTicketSource{searchErr: errors.New("boom")}, nil, nil, nil)
	_, err := svc.Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
	require.ErrorContains(t, err, "find tickets")
}

func TestFind_DocSearchFailure_FailsTheAnswer(t *testing.T) {
	svc := newTestService(t, nil, &fakeDocSource{searchErr: errors.New("boom")}, nil, nil)
	_, err := svc.Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
	require.ErrorContains(t, err, "find docs")
}

func TestFind_NamedSources(t *testing.T) {
	now := time.Now()
	svc := newTestService(t, nil, nil, nil, nil)
	svc.SetNamed(map[Kind]NamedSource{
		KindStack: fakeNamedSource{named: []Named{
			{ID: "s-old", Title: "deploy api", UpdatedAt: now.Add(-time.Hour)},
			{ID: "s-new", Title: "deploy", UpdatedAt: now},
			{ID: "s-redeploy", Title: "redeploy", UpdatedAt: now},
		}},
		KindAutomation: fakeNamedSource{err: apperrs.ErrForbidden},
	})

	found, err := svc.Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
	require.NoError(t, err)
	require.Len(t, found.Hits, 2, "a refused kind and a word inside another word find nothing")
	assert.Equal(t, "s-new", found.Hits[0].ID, "the title equal to the query leads")
	assert.Equal(t, "s-old", found.Hits[1].ID)
	assert.Empty(t, found.More)

	found, err = svc.Find(actorCtx("u-1"), "deploy", "ws-1", "stack", 1)
	require.NoError(t, err)
	require.Len(t, found.Hits, 1)
	assert.Equal(t, []string{"stack"}, found.More)

	svc.SetNamed(map[Kind]NamedSource{KindChannel: fakeNamedSource{err: errors.New("boom")}})
	_, err = svc.Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
	require.ErrorContains(t, err, "find channels")
}

func TestFind_TicketsAndDocs(t *testing.T) {
	now := time.Now()
	tickets := &fakeTicketSource{
		tickets: map[string]Ticket{
			"t-title": {ID: "t-title", Title: "Deploy logs", Status: "open", ProjectID: "p-1", Number: 4, UpdatedAt: now.Add(-time.Hour)},
			"t-body":  {ID: "t-body", Title: "Payments", Status: "open", ProjectID: "p-1", Number: 5, UpdatedAt: now},
			"t-key":   {ID: "t-key", Title: "Something else", Status: "open", ProjectID: "p-1", Number: 7},
		},
		search: []SearchHit{{ID: "t-body"}, {ID: "t-title"}, {ID: "t-gone"}},
		byKey:  map[string][]Ticket{"WEB-7": {{ID: "t-key", Title: "Something else", ProjectID: "p-1"}}},
	}
	docs := &fakeDocSource{
		docs:   map[string]Doc{"d-open": {ID: "d-open", Title: "Deploy guide"}, "d-locked": {ID: "d-locked", Title: "Deploy secrets"}},
		search: []SearchHit{{ID: "d-locked"}, {ID: "d-open"}, {ID: "d-gone"}},
	}
	access := &fakeAccessChecker{canOpen: map[string]bool{"d-open": true, "d-gone": true}}
	svc := newTestService(t, tickets, docs, &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open", Kind: "progress"}}}, access)
	svc.cfg.Projects = &fakeProjectSource{projects: map[string]Project{"p-1": {ID: "p-1", Prefix: "WEB"}}}

	found, err := svc.Find(actorCtx("u-1"), "deploy", "ws-1", "", 5)
	require.NoError(t, err)
	assert.Equal(t, "ws-1", tickets.searchWorkspace)
	require.Len(t, found.Hits, 3)
	assert.Equal(t, Hit{Type: "ticket", ID: "t-title", Title: "Deploy logs", Hint: "WEB-4", StatusLabel: "Open", StatusKind: "progress", ProjectID: "p-1", UpdatedAt: now.Add(-time.Hour)}, found.Hits[0])
	assert.True(t, found.Hits[1].InBody, "a ticket whose title lacks the word was found in its body")
	assert.Equal(t, "d-open", found.Hits[2].ID, "a doc the caller cannot open is left out")

	found, err = svc.Find(actorCtx("u-1"), "WEB-7", "ws-1", "ticket", 5)
	require.NoError(t, err)
	require.NotEmpty(t, found.Hits)
	assert.Equal(t, "t-key", found.Hits[0].ID, "the ticket a key names leads")
	assert.False(t, found.Hits[0].InBody)
}
