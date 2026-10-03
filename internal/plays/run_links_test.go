package plays

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLinks struct {
	links TicketLinks
	err   error
	asked []string
}

func (f *fakeLinks) TicketLinks(_ context.Context, id string) (TicketLinks, error) {
	f.asked = append(f.asked, id)
	return f.links, f.err
}

func runnerWithLinks(f *runnerFixture, links *fakeLinks) {
	f.runner.links = links
}

func TestRun_BugPlay_NamesItsOriginOneHop(t *testing.T) {
	f := newRunnerFixture()
	runnerWithLinks(f, &fakeLinks{links: TicketLinks{Origin: &LinkedTicket{Key: "NEX-7", Title: "Login page", Done: true}}})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, []string{`This bug was found in NEX-7 "Login page", a ticket whose work is recorded; fix the bug without rewriting that record. ` +
		"Read it with ticket_get for its body, doc, and pull requests, one hop only."}, f.turns.last().Play.Blocks)
}

func TestRun_BugPlay_OriginUnknown_SaysSo(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = nil
	runnerWithLinks(f, &fakeLinks{links: TicketLinks{OriginUnknown: true}})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	blocks := f.turns.last().Play.Blocks
	require.Len(t, blocks, 1)
	assert.Contains(t, blocks[0], "origin is unknown")
	assert.Contains(t, blocks[0], "Do not guess one")
}

func TestRun_BlockedTicket_ListsEachBlockerAndWhetherDone(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = nil
	runnerWithLinks(f, &fakeLinks{links: TicketLinks{Blockers: []LinkedTicket{
		{Key: "NEX-3", Title: "backend /books", Done: false},
		{Key: "NEX-4", Title: "schema", Done: true},
	}}})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	blocks := f.turns.last().Play.Blocks
	require.Len(t, blocks, 1)
	assert.Equal(t, "This ticket is blocked by these tickets; the user chose to run the play before they are all done:\n"+
		`- NEX-3 "backend /books": not done`+"\n"+
		`- NEX-4 "schema": done`, blocks[0])
}

func TestRun_LinkReadFails_RefusesWithoutTrail(t *testing.T) {
	f := newRunnerFixture()
	runnerWithLinks(f, &fakeLinks{err: errors.New("db down")})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.ErrorContains(t, err, "db down")
	assert.Empty(t, f.trails.all())
}

func TestRun_DocPlay_ReadsNoLinks(t *testing.T) {
	f := newRunnerFixture()
	links := &fakeLinks{err: errors.New("must not be read")}
	runnerWithLinks(f, links)

	_, err := f.runner.Run(ctxAs(starter), RunInput{PlayID: docPlayID, TargetType: TargetDoc, TargetID: docID})
	require.NoError(t, err)
	<-f.turns.done
	assert.Empty(t, links.asked)
}

func TestRenderLinkBlocks(t *testing.T) {
	assert.Empty(t, renderLinkBlocks(TicketLinks{}))

	allDone := renderLinkBlocks(TicketLinks{Blockers: []LinkedTicket{{Key: "NEX-4", Title: "schema", Done: true}}})
	require.Len(t, allDone, 1)
	assert.True(t, strings.HasPrefix(allDone[0], "This ticket was blocked by these tickets, all now done:"))
}
