package plays

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)

const workspaceID = "workspace-1"

func newTestService(repo Repo, perm PermissionGate) *Service {
	s := NewService(repo, perm)
	s.now = func() time.Time { return fixedNow }
	return s
}

func ctxAs(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func ticketStage() *Stage {
	s := StageProgress
	return &s
}

func TestCreate_TicketPlayWithoutStage_ReturnsValidationError(t *testing.T) {
	s := newTestService(newFakeRepo(), allowAll("owner"))
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestCreate_DocPlayWithStage_ReturnsValidationError(t *testing.T) {
	s := newTestService(newFakeRepo(), allowAll("owner"))
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc, ShowWhenStage: ticketStage()})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestCreate_EmptyLabel_ReturnsValidationError(t *testing.T) {
	s := newTestService(newFakeRepo(), allowAll("owner"))
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "  ", Type: TypeDoc})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestCreate_WithoutPlaysWrite_ReturnsForbidden(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakePerm(nil))
	_, err := s.Create(ctxAs("alice"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestCreate_ValidTicketPlay_PersistsAndPublishes(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{
		Label: "Fix with AI", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true,
	})
	require.NoError(t, err)
	assert.Equal(t, workspaceID, p.WorkspaceID)
	assert.Equal(t, "owner", p.CreatedBy)
	assert.Equal(t, fixedNow, p.CreatedAt)
	require.Len(t, repo.published, 1)
	assert.Equal(t, TopicCreated, repo.published[0].Topic)
}

func TestList_SortsByLabel(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Zeta", Type: TypeDoc})
	require.NoError(t, err)
	_, err = s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Alpha", Type: TypeDoc})
	require.NoError(t, err)

	list, err := s.List(ctxAs("owner"), workspaceID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// The fake repo isn't sort-aware; the real SQL query (ORDER BY label) is covered by the storage integration test.
	labels := map[string]bool{list[0].Label: true, list[1].Label: true}
	assert.True(t, labels["Zeta"] && labels["Alpha"])
}

func TestList_WithoutPlaysRead_ReturnsForbidden(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakePerm(nil))
	_, err := s.List(ctxAs("alice"), workspaceID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestGet_FromAnotherWorkspace_ReturnsNotFound(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.NoError(t, err)

	_, err = s.Get(ctxAs("owner"), "other-workspace", p.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUpdate_LeavesTypeUnchanged(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.NoError(t, err)

	updated, err := s.Update(ctxAs("owner"), workspaceID, p.ID, UpdateInput{Label: "Doc renamed", Enabled: false})
	require.NoError(t, err)
	assert.Equal(t, TypeDoc, updated.Type)
	assert.Equal(t, "Doc renamed", updated.Label)
	assert.False(t, updated.Enabled)
}

func TestUpdate_TicketPlayDroppingStage_ReturnsValidationError(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage()})
	require.NoError(t, err)

	_, err = s.Update(ctxAs("owner"), workspaceID, p.ID, UpdateInput{Label: "Fix", ShowWhenStage: nil})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestUpdate_WithoutPlaysWrite_ReturnsForbidden(t *testing.T) {
	repo := newFakeRepo()
	seed := newTestService(repo, allowAll("owner"))
	p, err := seed.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.NoError(t, err)

	s := newTestService(repo, newFakePerm(nil))
	_, err = s.Update(ctxAs("alice"), workspaceID, p.ID, UpdateInput{Label: "Doc"})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestDelete_RemovesPlayAndPublishes(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	p, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.NoError(t, err)

	err = s.Delete(ctxAs("owner"), workspaceID, p.ID)
	require.NoError(t, err)
	_, err = s.Get(ctxAs("owner"), workspaceID, p.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.Len(t, repo.published, 2) // created + deleted
	assert.Equal(t, TopicDeleted, repo.published[1].Topic)
}

func TestDelete_WithoutPlaysDelete_ReturnsForbidden(t *testing.T) {
	repo := newFakeRepo()
	seed := newTestService(repo, allowAll("owner"))
	p, err := seed.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Doc", Type: TypeDoc})
	require.NoError(t, err)

	s := newTestService(repo, newFakePerm(map[string][]permissions.Action{"alice": {permissions.PlaysRead}}))
	err = s.Delete(ctxAs("alice"), workspaceID, p.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestSeedDefaults_CreatesFixWithAIToTicketsViaAIAndInterview(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakePerm(nil)) // no permission gate needed; SeedDefaults bypasses it

	err := s.SeedDefaults(context.Background(), workspaceID)
	require.NoError(t, err)

	list, err := repo.List(context.Background(), workspaceID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	byLabel := map[string]*Play{}
	for _, p := range list {
		byLabel[p.Label] = p
	}
	fix := byLabel["Fix with AI"]
	require.NotNil(t, fix)
	assert.Equal(t, TypeTicket, fix.Type)
	require.NotNil(t, fix.ShowWhenStage)
	assert.Equal(t, StageProgress, *fix.ShowWhenStage)
	assert.True(t, fix.Enabled)

	toTickets := byLabel["To tickets via AI"]
	require.NotNil(t, toTickets)
	assert.Equal(t, TypeDoc, toTickets.Type)
	assert.Nil(t, toTickets.ShowWhenStage)

	interview := byLabel["Interview"]
	require.NotNil(t, interview)
	assert.Equal(t, TypeInterview, interview.Type)
	assert.Nil(t, interview.ShowWhenStage)
	assert.True(t, interview.Enabled)
	assert.Contains(t, interview.Instructions, "memory_create_interview")
	assert.Contains(t, interview.Instructions, "one question at a time")
}

func TestNormalizeProjectIDs_TrimsDropsEmptyDedupesAndSorts(t *testing.T) {
	got := normalizeIDs([]string{" proj-2 ", "proj-1", "", "proj-1", "  "})
	assert.Equal(t, []string{"proj-1", "proj-2"}, got)
}

func TestGet_EmptyID_ReturnsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo(), allowAll("owner"))
	_, err := s.Get(ctxAs("owner"), workspaceID, "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestActorID_NoActorInContext_ReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", actorID(context.Background()))
}

// runnerPerm grants owner the write/read/delete bits Create needs to seed fixtures, plus plays:run for
// whichever users are named, so ListApplicable tests can then deny specific plays per user.
func runnerPerm(runners ...string) *fakePerm {
	grants := map[string][]permissions.Action{
		"owner": {permissions.PlaysWrite, permissions.PlaysRead, permissions.PlaysDelete},
	}
	for _, u := range runners {
		grants[u] = append(grants[u], permissions.PlaysRun)
	}
	return newFakePerm(grants)
}

func TestListApplicable_TicketPlay_MatchesEnabledStageAndPermission(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	fix, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true})
	require.NoError(t, err)

	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, fix.ID, list[0].ID)
}

func TestListApplicable_InterviewPlay_ListsOnlyForTheInterviewType(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, runnerPerm("alice"))
	interview, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Interview", Type: TypeInterview, Enabled: true})
	require.NoError(t, err)

	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeInterview, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, interview.ID, list[0].ID)

	docs, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeDoc, nil)
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestListApplicable_DisabledPlay_NeverLists(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: false})
	require.NoError(t, err)

	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestListApplicable_ExcludedProject_NeverLists(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{
		Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true, ExcludedProjectIDs: []string{"proj-1"},
	})
	require.NoError(t, err)

	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	assert.Empty(t, list)

	other, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-2", TypeTicket, &stage)
	require.NoError(t, err)
	assert.Len(t, other, 1, "a different project is unaffected by the exclusion")
}

func TestListApplicable_WrongStage_NeverListsForATicketPlay(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true})
	require.NoError(t, err)

	wrong := StageReview
	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, &wrong)
	require.NoError(t, err)
	assert.Empty(t, list)

	list, err = s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, nil)
	require.NoError(t, err)
	assert.Empty(t, list, "no stage at all never matches a ticket play")
}

func TestListApplicable_DeniedUser_Omits(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	fix, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true})
	require.NoError(t, err)
	perm.deny("alice", fix.ID)

	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestListApplicable_UserWithoutPlaysRun_Omits(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm() // nobody granted plays:run
	s := newTestService(repo, perm)
	_, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true})
	require.NoError(t, err)

	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "bob", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestListApplicable_DocPlay_IgnoresStage(t *testing.T) {
	repo := newFakeRepo()
	perm := runnerPerm("alice")
	s := newTestService(repo, perm)
	toTickets, err := s.Create(ctxAs("owner"), workspaceID, CreateInput{Label: "To tickets", Type: TypeDoc, Enabled: true})
	require.NoError(t, err)

	list, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", TypeDoc, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, toTickets.ID, list[0].ID)
}

func TestListApplicable_OwnerBypassIgnoresTheDenial(t *testing.T) {
	repo := newFakeRepo()
	fix, err := newTestService(repo, runnerPerm()).Create(
		ctxAs("owner"), workspaceID, CreateInput{Label: "Fix", Type: TypeTicket, ShowWhenStage: ticketStage(), Enabled: true},
	)
	require.NoError(t, err)

	// ownerBypassPerm mirrors access.Service.HasPermission's Owner-role bypass ahead of any resource-instance deny.
	s := newTestService(repo, ownerBypassPerm{})
	stage := StageProgress
	list, err := s.ListApplicable(context.Background(), workspaceID, "owner", "proj-1", TypeTicket, &stage)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, fix.ID, list[0].ID)
}

// ownerBypassPerm always allows, mirroring access.Service.HasPermission's Owner-role bypass ahead of any
// resource-instance deny (ticket 21's requirement that Owner ignores a play exclusion).
type ownerBypassPerm struct{}

func (ownerBypassPerm) HasPermission(context.Context, string, string, permissions.Action, string, string) bool {
	return true
}

func TestListApplicable_InvalidType_ReturnsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo(), runnerPerm("alice"))
	_, err := s.ListApplicable(context.Background(), workspaceID, "alice", "proj-1", Type("bogus"), nil)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestListApplicable_EmptyWorkspaceID_ReturnsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo(), runnerPerm("alice"))
	_, err := s.ListApplicable(context.Background(), "", "alice", "proj-1", TypeTicket, nil)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestSeedDefaults_AlreadySeeded_IsANoOp(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakePerm(nil))
	require.NoError(t, s.SeedDefaults(context.Background(), workspaceID))
	require.NoError(t, s.SeedDefaults(context.Background(), workspaceID))

	list, err := repo.List(context.Background(), workspaceID)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}
