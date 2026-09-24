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

func TestRun_BugPlay_CarriesOneHopOfOriginContext(t *testing.T) {
	f := newRunnerFixture()
	runnerWithLinks(f, &fakeLinks{links: TicketLinks{Origin: &OriginContext{
		LinkedTicket: LinkedTicket{Key: "NEX-7", Title: "Login page", Done: true},
		Body:         "## Acceptance criteria\nlogs in",
		DocTitle:     "Auth spec",
		DocBody:      "sessions last a day",
		PRs:          []PullRequest{{Owner: "otal", Repo: "nexul", Number: 12, Title: "Add login", State: "merged"}},
	}}})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	blocks := f.turns.last().ExtraRequestBlocks
	require.Len(t, blocks, 2, "the play, then the origin")
	origin := blocks[1]
	assert.Contains(t, origin, `This bug was found in NEX-7 "Login page"`)
	assert.Contains(t, origin, "Origin ticket body:\n## Acceptance criteria\nlogs in")
	assert.Contains(t, origin, "Origin doc \"Auth spec\":\nsessions last a day")
	assert.Contains(t, origin, `- otal/nexul#12 "Add login" (merged)`)
}

func TestRun_BugPlay_OriginUnknown_SaysSo(t *testing.T) {
	f := newRunnerFixture()
	f.mems.byProject[projectID] = nil
	runnerWithLinks(f, &fakeLinks{links: TicketLinks{OriginUnknown: true}})

	_, err := f.runner.Run(ctxAs(starter), ticketRun())
	require.NoError(t, err)
	<-f.turns.done

	blocks := f.turns.last().ExtraRequestBlocks
	require.Len(t, blocks, 2)
	assert.Contains(t, blocks[1], "origin is unknown")
	assert.Contains(t, blocks[1], "Do not guess one")
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

	blocks := f.turns.last().ExtraRequestBlocks
	require.Len(t, blocks, 2)
	assert.Equal(t, "This ticket is blocked by these tickets; the user chose to run the play before they are all done:\n"+
		`- NEX-3 "backend /books": not done`+"\n"+
		`- NEX-4 "schema": done`, blocks[1])
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

	bare := renderLinkBlocks(TicketLinks{Origin: &OriginContext{LinkedTicket: LinkedTicket{Key: "NEX-7", Title: "Login"}}})
	require.Len(t, bare, 1)
	assert.Contains(t, bare[0], "Origin ticket body:\n(empty)")
	assert.Contains(t, bare[0], "Origin pull requests: (none)")
	assert.NotContains(t, bare[0], "Origin doc")

	huge := renderOrigin(OriginContext{Body: strings.Repeat("é", originCharCap)})
	assert.Contains(t, huge, "(trimmed to fit the turn size limit)")
	assert.Less(t, len(huge), originCharCap+500)
}
