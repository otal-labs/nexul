package plays

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func rule(field Field, op Op, values ...string) Rule {
	return Rule{Field: field, Op: op, Values: values}
}

func when(rules ...Rule) Group {
	return Group{Match: MatchAll, Rules: rules}
}

func stagePtr(s Stage) *Stage { return &s }

func TestAutoPlayValidate_RefusesEveryInvalidShape(t *testing.T) {
	tooMany := make([]Rule, MaxAutoPlayRules+1)
	for i := range tooMany {
		tooMany[i] = rule(FieldBlocked, OpSet)
	}
	tooManyValues := make([]string, MaxAutoPlayRuleValues+1)
	for i := range tooManyValues {
		tooManyValues[i] = string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	tests := []struct {
		name     string
		playType Type
		a        AutoPlay
		want     string
	}{
		{"an interview play", TypeInterview, AutoPlay{Moment: MomentTicketCreated}, "interview play takes no auto plays"},
		{"an unknown moment", TypeTicket, AutoPlay{Moment: "ticket.moved"}, "moment must be one of"},
		{"a doc moment on a ticket play", TypeTicket, AutoPlay{Moment: MomentDocChanged}, "belongs to doc plays"},
		{"a ticket moment on a doc play", TypeDoc, AutoPlay{Moment: MomentTicketUnblocked}, "belongs to ticket plays"},
		{"entering a stage with no stage", TypeTicket, AutoPlay{Moment: MomentTicketEnteredStage}, "needs moment_stage"},
		{"entering a stage that does not exist", TypeTicket, AutoPlay{Moment: MomentTicketEnteredStage, MomentStage: stagePtr("shipped")}, "needs moment_stage"},
		{"an unknown run_on", TypeTicket, AutoPlay{Moment: MomentTicketCreated, RunOn: "owner"}, "run_on must be"},
		{"a doc play run on its developer", TypeDoc, AutoPlay{Moment: MomentDocCreated, RunOn: RunOnDeveloper}, "runs on causer"},
		{"a negative limit", TypeTicket, AutoPlay{Moment: MomentTicketCreated, OnceWithinMinutes: -1}, "once_within_minutes"},
		{"a limit past 30 days", TypeTicket, AutoPlay{Moment: MomentTicketCreated, OnceWithinMinutes: MaxOnceWithinMinutes + 1}, "once_within_minutes"},
		{"a top match that is neither", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Match: "most"}}, "conditions match"},
		{"a group match that is neither", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{{Match: "most", Rules: []Rule{rule(FieldBlocked, OpSet)}}}}}, "match must be all or any"},
		{"an empty group", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{{}}}}, "has no rules"},
		{"a doc field on a ticket play", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldFolder, OpIs, "f-1"))}}}, `field "folder" is not one a ticket play reads`},
		{"a ticket field on a doc play", TypeDoc, AutoPlay{Moment: MomentDocCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldLabel, OpIs, "bug"))}}}, `field "label" is not one a doc play reads`},
		{"an op the field does not take", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldBlocked, OpIs, "yes"))}}}, "takes the ops set, unset"},
		{"set with values", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldDeveloper, OpSet, "u-1"))}}}, "set takes no values"},
		{"is with no values", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldType, OpIs, " "))}}}, "is needs at least one value"},
		{"too many values", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldLabel, OpIs, tooManyValues...))}}}, "at most 50 values"},
		{"a stage that does not exist", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldStage, OpIs, "shipped"))}}}, `stage "shipped" is not one of`},
		{"an unknown otherwise", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Priority: Priority{Otherwise: "urgent"}}, "priority otherwise"},
		{"an unknown rule level", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Priority: Priority{Rules: []PriorityRule{{Level: "urgent", When: when(rule(FieldBlocked, OpSet))}}}}, "level must be"},
		{"a priority rule with an empty group", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Priority: Priority{Rules: []PriorityRule{{Level: LevelHigh}}}}, "priority rule 1 has no rules"},
		{"more rules than the cap", TypeTicket, AutoPlay{Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(tooMany[:10]...)}}, Priority: Priority{Rules: []PriorityRule{{Level: LevelHigh, When: when(tooMany[10:]...)}}}}, "at most 20 rules"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.a
			a.normalize(tt.playType)
			err := a.Validate(tt.playType)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestAutoPlayValidate_AcceptsWhatTheComposerBuilds(t *testing.T) {
	tests := []struct {
		name     string
		playType Type
		a        AutoPlay
	}{
		{"a bare moment", TypeTicket, AutoPlay{Moment: MomentTicketUnblocked}},
		{"a stage moment", TypeTicket, AutoPlay{Moment: MomentTicketEnteredStage, MomentStage: stagePtr(StageDone), RunOn: RunOnCauser}},
		{"a doc moment", TypeDoc, AutoPlay{Moment: MomentDocChanged, Conditions: Conditions{Match: MatchAny, Groups: []Group{when(rule(FieldFolder, OpIsNot, "f-1"))}}}},
		{"two levels of conditions and a priority", TypeTicket, AutoPlay{
			Moment: MomentTicketTestFailed, RunOn: RunOnTester, OnceWithinMinutes: 60,
			Conditions: Conditions{Match: MatchAll, Groups: []Group{
				when(rule(FieldType, OpIs, "Bug"), rule(FieldStage, OpIsNot, "done")),
				{Match: MatchAny, Rules: []Rule{rule(FieldLinkedPR, OpSet), rule(FieldSourceDoc, OpUnset)}},
			}},
			Priority: Priority{Rules: []PriorityRule{{Level: LevelHigh, When: when(rule(FieldLabel, OpIs, "urgent"))}}, Otherwise: LevelLow},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.a
			a.normalize(tt.playType)
			require.NoError(t, a.Validate(tt.playType))
		})
	}
}

func TestAutoPlayNormalize_FillsWhatAnOmittedPartMeans(t *testing.T) {
	a := AutoPlay{
		Moment: MomentTicketCreated, MomentStage: stagePtr(StageDone),
		Conditions: Conditions{Groups: []Group{{Rules: []Rule{rule(FieldLabel, OpIs, " bug ", "bug", "")}}}},
	}
	a.normalize(TypeTicket)
	assert.Equal(t, MatchAll, a.Conditions.Match)
	assert.Equal(t, MatchAll, a.Conditions.Groups[0].Match)
	assert.Equal(t, []string{"bug"}, a.Conditions.Groups[0].Rules[0].Values, "trimmed and deduped")
	assert.Equal(t, Priority{Rules: []PriorityRule{}, Otherwise: LevelNormal}, a.Priority)
	assert.Equal(t, RunOnDeveloper, a.RunOn)
	assert.Nil(t, a.MomentStage, "a stage only belongs to ticket.entered_stage")

	doc := AutoPlay{Moment: MomentDocCreated}
	doc.normalize(TypeDoc)
	assert.Equal(t, RunOnCauser, doc.RunOn, "a doc has no developer, so its auto plays run on whoever caused the moment")
	assert.Equal(t, []Group{}, doc.Conditions.Groups)
}

// autoPlayFixture seeds a ticket play, a doc play, and an interview play; "editor" holds every autoplays bit.
func autoPlayFixture(t *testing.T) (*Service, *fakeRepo, map[Type]*Play) {
	t.Helper()
	repo := newFakeRepo()
	perm := newFakePerm(map[string][]permissions.Action{
		"owner":  {permissions.PlaysWrite},
		"editor": {permissions.PlaysRead, permissions.AutoplaysRead, permissions.AutoplaysWrite, permissions.AutoplaysDelete},
		"player": {permissions.PlaysRead, permissions.PlaysWrite, permissions.PlaysDelete},
		"reader": {permissions.AutoplaysRead},
		"writer": {permissions.AutoplaysWrite},
	})
	s := newTestService(repo, perm)
	out := map[Type]*Play{}
	for _, in := range []CreateInput{
		{Label: "Fix with AI", Type: TypeTicket, ShowWhenStage: ticketStage()},
		{Label: "To tickets", Type: TypeDoc},
		{Label: "Interview", Type: TypeInterview},
	} {
		p, err := s.Create(ctxAs("owner"), workspaceID, in)
		require.NoError(t, err)
		out[p.Type] = p
	}
	repo.published = nil
	return s, repo, out
}

func TestAutoPlays_EachVerbNeedsItsOwnBit(t *testing.T) {
	s, _, ps := autoPlayFixture(t)
	existing, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.NoError(t, err)
	playID := ps[TypeTicket].ID
	verbs := map[string]func(user string) error{
		"list": func(u string) error { _, err := s.ListAutoPlays(ctxAs(u), workspaceID, playID); return err },
		"list by play": func(u string) error {
			_, err := s.AutoPlaysByPlay(ctxAs(u), workspaceID, []string{playID})
			return err
		},
		"create": func(u string) error {
			_, err := s.CreateAutoPlay(ctxAs(u), workspaceID, playID, AutoPlayInput{Moment: MomentTicketCreated})
			return err
		},
		"update": func(u string) error {
			_, err := s.UpdateAutoPlay(ctxAs(u), workspaceID, playID, existing.ID, true, AutoPlayInput{Moment: MomentTicketCreated})
			return err
		},
		"delete":   func(u string) error { return s.DeleteAutoPlay(ctxAs(u), workspaceID, playID, existing.ID) },
		"read cap": func(u string) error { _, err := s.AutoPlayDailyCap(ctxAs(u), workspaceID); return err },
		"set cap":  func(u string) error { _, err := s.SetAutoPlayDailyCap(ctxAs(u), workspaceID, 3); return err },
	}
	denied := map[string][]string{
		"player": {"list", "list by play", "create", "update", "delete", "read cap", "set cap"},
		"reader": {"create", "update", "delete", "set cap"},
		"writer": {"list", "list by play", "delete", "read cap"},
	}
	for user, names := range denied {
		for _, name := range names {
			err := verbs[name](user)
			assert.ErrorIs(t, err, apperrs.ErrForbidden, "%s as %s", name, user)
		}
	}
}

func TestCreateAutoPlay_StartsOffAndPublishes(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketUnblocked})
	require.NoError(t, err)
	assert.False(t, a.Enabled, "a new auto play waits for someone to turn it on")
	assert.Equal(t, ps[TypeTicket].ID, a.PlayID)
	assert.Equal(t, workspaceID, a.WorkspaceID)
	assert.Equal(t, "editor", a.CreatedBy)
	require.Len(t, repo.published, 1)
	assert.Equal(t, TopicAutoPlayCreated, repo.published[0].Topic)
	assert.Equal(t, AutoPlayEvent{AutoPlay: *a}, repo.published[0].Payload)
}

func TestCreateAutoPlay_Refusals(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeInterview].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "an interview play takes none")
	_, err = s.CreateAutoPlay(ctxAs("editor"), "workspace-2", ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a play of another workspace is not found")
	repo.autoPlayErr = errors.New("disk full")
	_, err = s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorContains(t, err, "disk full")
	assert.Empty(t, repo.published)
}

func TestUpdateAutoPlay_SwitchesOnAndReplacesTheTrees(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{
		Moment: MomentTicketCreated, Conditions: Conditions{Groups: []Group{when(rule(FieldBlocked, OpUnset))}},
	})
	require.NoError(t, err)
	updated, err := s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID, true, AutoPlayInput{
		Moment: MomentTicketEnteredStage, MomentStage: stagePtr(StageReview), RunOn: RunOnTester,
	})
	require.NoError(t, err)
	assert.True(t, updated.Enabled)
	assert.Equal(t, []Group{}, updated.Conditions.Groups, "conditions are replaced whole")
	assert.Equal(t, StageReview, *updated.MomentStage)
	assert.Equal(t, TopicAutoPlayUpdated, repo.published[len(repo.published)-1].Topic)

	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID, true, AutoPlayInput{Moment: MomentDocChanged})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeDoc].ID, a.ID, true, AutoPlayInput{Moment: MomentDocChanged})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an auto play under another play is not found")
	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, " ", true, AutoPlayInput{Moment: MomentTicketCreated})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	repo.autoPlayErr = errors.New("disk full")
	_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID, false, AutoPlayInput{Moment: MomentTicketCreated})
	require.Error(t, err)
}

func TestDeleteAutoPlay_PublishesWhatWasDeleted(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.NoError(t, err)
	require.ErrorIs(t, s.DeleteAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, "missing"), apperrs.ErrNotFound)
	require.NoError(t, s.DeleteAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, a.ID))
	last := repo.published[len(repo.published)-1]
	assert.Equal(t, TopicAutoPlayDeleted, last.Topic)
	assert.Equal(t, AutoPlayDeletedEvent{ID: a.ID, PlayID: ps[TypeTicket].ID, WorkspaceID: workspaceID}, last.Payload)
	list, err := s.ListAutoPlays(ctxAs("editor"), workspaceID, ps[TypeTicket].ID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestAutoPlaysByPlay_OneReadAndAnEntryPerPlay(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.NoError(t, err)
	byPlay, err := s.AutoPlaysByPlay(ctxAs("editor"), workspaceID, []string{ps[TypeTicket].ID, ps[TypeDoc].ID})
	require.NoError(t, err)
	assert.Len(t, byPlay[ps[TypeTicket].ID], 1)
	assert.Equal(t, []*AutoPlay{}, byPlay[ps[TypeDoc].ID])
	assert.Equal(t, 1, repo.autoPlayLists)

	repo.autoPlayErr = errors.New("disk full")
	_, err = s.AutoPlaysByPlay(ctxAs("editor"), workspaceID, []string{ps[TypeTicket].ID})
	require.Error(t, err)
	_, err = s.ListAutoPlays(ctxAs("editor"), workspaceID, ps[TypeTicket].ID)
	require.Error(t, err)
}

func TestAutoPlayDailyCap_StaysInItsBounds(t *testing.T) {
	s, repo, _ := autoPlayFixture(t)
	limit, err := s.AutoPlayDailyCap(ctxAs("editor"), workspaceID)
	require.NoError(t, err)
	assert.Equal(t, DefaultAutoPlayDailyCap, limit)
	for _, bad := range []int{0, MaxAutoPlayDailyCap + 1} {
		_, err := s.SetAutoPlayDailyCap(ctxAs("editor"), workspaceID, bad)
		require.ErrorIs(t, err, apperrs.ErrInvalid, "%d", bad)
	}
	repo.published = nil
	limit, err = s.SetAutoPlayDailyCap(ctxAs("editor"), workspaceID, 12)
	require.NoError(t, err)
	assert.Equal(t, 12, limit)
	require.Len(t, repo.published, 1)
	assert.Equal(t, TopicAutoPlayLimitsUpdated, repo.published[0].Topic)
	assert.Equal(t, AutoPlayLimitsEvent{WorkspaceID: workspaceID, DailyCapPerTicket: 12}, repo.published[0].Payload)

	repo.autoPlayErr = errors.New("disk full")
	_, err = s.AutoPlayDailyCap(ctxAs("editor"), workspaceID)
	require.Error(t, err)
	_, err = s.SetAutoPlayDailyCap(ctxAs("editor"), workspaceID, 3)
	require.Error(t, err)
}
