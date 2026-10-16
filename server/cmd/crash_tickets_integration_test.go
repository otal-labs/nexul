package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func crashEvent(t *testing.T, topic, stackID, outageID string) eventbus.Event {
	t.Helper()
	ended := time.Date(2026, 10, 16, 10, 45, 0, 0, time.UTC)
	p := deploy.OutageEvent{
		WorkspaceID: "workspace-default", ProjectID: "project-general", StackID: stackID, StackName: "web", Service: "web", Container: "web", Machine: "m1",
		OutageID: outageID, ExitCode: 137, OOM: true, StartedAt: time.Date(2026, 10, 16, 10, 42, 0, 0, time.UTC),
	}
	if topic == deploy.TopicContainerRecovered {
		p.EndedAt = &ended
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	return eventbus.Event{ID: outageID + topic, Topic: topic, Payload: raw}
}

func switchCrashTickets(t *testing.T, f permFixture, stackID string, on bool) {
	t.Helper()
	stack, err := f.svc.deploySvc.GetStack(as(uOwner), stackID)
	require.NoError(t, err)
	stack.CrashTickets, stack.DockerNetwork = on, "net"
	_, err = f.svc.deploySvc.UpdateStack(as(uOwner), *stack)
	require.NoError(t, err)
}

func projectTickets(t *testing.T, f permFixture) []*tickets.Ticket {
	t.Helper()
	ts, err := f.svc.ticketsSvc.ListByProject(as(uOwner), "project-general")
	require.NoError(t, err)
	return ts
}

func threadBodies(t *testing.T, f permFixture, ticketID string) []string {
	t.Helper()
	thread, err := f.svc.chatSvc.ExistingThread(as(uOwner), chat.KindTicketThread, ticketID, uOwner)
	require.NoError(t, err)
	ms, err := f.svc.chatSvc.ListMessages(as(uOwner), thread.ID, uOwner, 50)
	require.NoError(t, err)
	var out []string
	for _, m := range ms {
		assert.Equal(t, chat.AuthorSystem, m.AuthorKind)
		out = append(out, m.Body)
	}
	return out
}

// TestCrashTickets_ASwitchedOnStackFilesOneTicketAndCommentsOnTheRest walks a stack's crashes through the wired
// consumers: nothing while the switch is off, one ticket for the first crash, comments for a repeat and a recovery, and
// a fresh ticket once the earlier one is done.
func TestCrashTickets_ASwitchedOnStackFilesOneTicketAndCommentsOnTheRest(t *testing.T) {
	f := newPermFixture(t)
	withMachineM1(t, f)
	d := f.svc.deploySvc
	ctx := context.Background()
	before := len(projectTickets(t, f))

	require.NoError(t, d.HandleContainerStopped(ctx, crashEvent(t, deploy.TopicContainerStopped, f.stack, "o1")))
	assert.Len(t, projectTickets(t, f), before, "the switch is off by default")

	switchCrashTickets(t, f, f.stack, true)
	require.NoError(t, d.HandleContainerStopped(ctx, crashEvent(t, deploy.TopicContainerStopped, f.stack, "o1")))
	ts := projectTickets(t, f)
	require.Len(t, ts, before+1)
	filed := ts[len(ts)-1]
	assert.Equal(t, "web crashed: exit 137, out of memory", filed.Title)
	assert.Contains(t, filed.Body, "stack_get")
	assert.Empty(t, filed.Developer)
	assert.Empty(t, filed.Tester)
	assert.Empty(t, filed.Reporter.Login, "created by the instance, no person behind it")
	first, err := f.store.Statuses.FirstStatus(ctx, "project-general")
	require.NoError(t, err)
	assert.Equal(t, tickets.Status(first), filed.Status, "the project's first column")
	firstType, err := f.store.TicketTypes.FirstType(ctx, "project-general")
	require.NoError(t, err)
	assert.Equal(t, firstType, filed.TypeID, "the project's default type")

	require.NoError(t, d.HandleContainerStopped(ctx, crashEvent(t, deploy.TopicContainerStopped, f.stack, "o2")))
	require.NoError(t, d.HandleContainerRecovered(ctx, crashEvent(t, deploy.TopicContainerRecovered, f.stack, "o2")))
	assert.Len(t, projectTickets(t, f), before+1, "a repeat and a recovery file nothing")
	bodies := threadBodies(t, f, filed.ID)
	require.Len(t, bodies, 2)
	assert.Contains(t, bodies[0], "crashed again")
	assert.Contains(t, bodies[1], "came back")

	statuses, err := f.store.Statuses.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	for _, st := range statuses {
		if st.Kind != workspace.StatusKindDone {
			continue
		}
		_, err := f.svc.ticketsSvc.UpdateStatus(as(uOwner), filed.ID, tickets.Status(st.ID))
		require.NoError(t, err)
		break
	}
	require.NoError(t, d.HandleContainerRecovered(ctx, crashEvent(t, deploy.TopicContainerRecovered, f.stack, "o2")))
	assert.Len(t, threadBodies(t, f, filed.ID), 2, "a done ticket gets no more comments")
	require.NoError(t, d.HandleContainerStopped(ctx, crashEvent(t, deploy.TopicContainerStopped, f.stack, "o3")))
	assert.Len(t, projectTickets(t, f), before+2, "a done ticket is not open, so the next crash files a new one")
}

// A bug type needs a found-in, which a crash cannot name, so the ticket is filed with its origin marked unknown.
func TestCrashTickets_ABugDefaultTypeIsFiledWithAnUnknownOrigin(t *testing.T) {
	f := newPermFixture(t)
	withMachineM1(t, f)
	_, err := f.db.Exec(`UPDATE ticket_types SET position = -1 WHERE id = 'ticket-type-bug'`)
	require.NoError(t, err)
	switchCrashTickets(t, f, f.stack, true)
	before := len(projectTickets(t, f))

	require.NoError(t, f.svc.deploySvc.HandleContainerStopped(context.Background(), crashEvent(t, deploy.TopicContainerStopped, f.stack, "o1")))

	ts := projectTickets(t, f)
	require.Len(t, ts, before+1)
	filed := ts[len(ts)-1]
	assert.Equal(t, "ticket-type-bug", filed.TypeID)
	links, err := f.svc.ticketsSvc.Links(as(uOwner), filed.ID)
	require.NoError(t, err)
	assert.True(t, links.OriginUnknown)
	assert.Nil(t, links.FoundIn)
}

func TestCrashTickets_AnInstanceStackCannotSwitchThemOnAndFilesNothing(t *testing.T) {
	f := newPermFixture(t)
	withMachineM1(t, f)
	stack, err := f.svc.deploySvc.GetStack(as(uOwner), f.infra)
	require.NoError(t, err)
	stack.CrashTickets, stack.DockerNetwork = true, "net"
	_, err = f.svc.deploySvc.UpdateStack(as(uOwner), *stack)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "crash tickets need a stack in a project")

	before := len(projectTickets(t, f))
	ev := crashEvent(t, deploy.TopicContainerStopped, f.infra, "o1")
	require.NoError(t, f.svc.deploySvc.HandleContainerStopped(context.Background(), ev))
	assert.Len(t, projectTickets(t, f), before)
}
