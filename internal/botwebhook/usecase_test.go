package botwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 10, 6, 21, 11, 0, 0, time.UTC)

// fakeRepo is an in-memory Repo; its live-name check is exact-case, so case folding is the use-case's to prove.
type fakeRepo struct {
	mu     sync.Mutex
	order  []string
	bots   map[string]*Bot
	events []eventbus.OutboxEvent

	getErr, listErr, createErr, updateErr error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{bots: map[string]*Bot{}} }

func (f *fakeRepo) nameTaken(b *Bot) bool {
	for _, o := range f.bots {
		if o.ID != b.ID && o.ConversationID == b.ConversationID && o.DeletedAt == nil && b.DeletedAt == nil && o.Name == b.Name {
			return true
		}
	}
	return false
}

func (f *fakeRepo) Create(_ context.Context, b *Bot, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if f.nameTaken(b) {
		return apperrs.ErrConflict
	}
	cp := *b
	f.bots[b.ID], f.order = &cp, append(f.order, b.ID)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*Bot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	b, ok := f.bots[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *b
	return &cp, nil
}

func (f *fakeRepo) List(_ context.Context, conversationID string, deleted bool) ([]*Bot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Bot
	for _, id := range f.order {
		b := f.bots[id]
		if b.ConversationID == conversationID && (b.DeletedAt != nil) == deleted {
			cp := *b
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, b *Bot, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	if _, ok := f.bots[b.ID]; !ok {
		return apperrs.ErrNotFound
	}
	if f.nameTaken(b) {
		return apperrs.ErrConflict
	}
	cp := *b
	f.bots[b.ID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

// fakeGate grants each user the actions listed for them, in every workspace.
type fakeGate struct {
	held map[string][]permissions.Action
	err  error
}

func (g fakeGate) Require(ctx context.Context, _ string, action permissions.Action) error {
	if g.err != nil {
		return g.err
	}
	actor, _ := identity.ActorFromCtx(ctx)
	for _, a := range g.held[actor.ID] {
		if a == action {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", apperrs.ErrForbidden, action)
}

type fakeConversations map[string]*Conversation

func (f fakeConversations) Conversation(_ context.Context, id string) (*Conversation, error) {
	c, ok := f[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return c, nil
}

type fakeInstance struct {
	url string
	err error
}

func (f fakeInstance) GetInstanceURL(context.Context) (string, error) { return f.url, f.err }

const (
	editor   = "u-alice"
	reader   = "u-bob"
	deleter  = "u-lena"
	everyone = "u-sam"
)

var all = []permissions.Action{permissions.BotwebhookRead, permissions.BotwebhookWrite, permissions.BotwebhookDelete}

func newTestService(repo *fakeRepo) *Service {
	s := NewService(Config{
		Repo: repo,
		Gate: fakeGate{held: map[string][]permissions.Action{
			editor:   {permissions.BotwebhookRead, permissions.BotwebhookWrite},
			reader:   {permissions.BotwebhookRead},
			deleter:  {permissions.BotwebhookDelete},
			everyone: all,
		}},
		Conversations: fakeConversations{
			"c-eng": {ID: "c-eng", WorkspaceID: "w-acme"},
			"c-dm":  {ID: "c-dm", WorkspaceID: "w-acme", MembersOnly: true},
		},
		Instance: fakeInstance{url: "https://nexul.example.com/"},
	})
	s.now = func() time.Time { return fixedNow }
	return s
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func mustCreate(t *testing.T, s *Service, name string) *Bot {
	t.Helper()
	b, err := s.Create(as(editor), "c-eng", name, "")
	require.NoError(t, err)
	return b
}

func tokenOf(t *testing.T, repo *fakeRepo, id string) string {
	t.Helper()
	b, err := repo.Get(context.Background(), id)
	require.NoError(t, err)
	return b.Token
}

func topics(evts []eventbus.OutboxEvent) []string {
	out := make([]string, len(evts))
	for i, e := range evts {
		out[i] = e.Topic
	}
	return out
}

func TestList_ReadWithoutWrite_HidesEveryURL(t *testing.T) {
	s := newTestService(newFakeRepo())
	mustCreate(t, s, "CI")

	bots, err := s.List(as(reader), "c-eng", false)
	require.NoError(t, err)
	require.Len(t, bots, 1)
	assert.Empty(t, bots[0].URL)
	raw, err := json.Marshal(bots)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "/api/botwebhooks/")

	_, err = s.List(as(reader), "c-eng", true)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the deleted list is for editors, who may restore")
	_, err = s.List(as(deleter), "c-eng", false)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "delete alone does not read")
}

func TestList_Write_ShowsTheURLOnTheInstanceAddress(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")

	bots, err := s.List(as(editor), "c-eng", false)
	require.NoError(t, err)
	require.Len(t, bots, 1)
	assert.Equal(t, "https://nexul.example.com/api/botwebhooks/"+b.ID+"/"+tokenOf(t, repo, b.ID), bots[0].URL)
	assert.Equal(t, bots[0].URL, b.URL, "create answers with the same URL the list shows")

	s.instance = fakeInstance{}
	bots, err = s.List(as(editor), "c-eng", false)
	require.NoError(t, err)
	assert.Equal(t, "/api/botwebhooks/"+b.ID+"/"+tokenOf(t, repo, b.ID), bots[0].URL, "before setup names the instance URL it is the path alone")
}

func TestCreate_WithoutWrite_IsForbidden(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(as(reader), "c-eng", "CI", "")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = s.Create(as(editor), "c-gone", "CI", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a conversation the caller cannot read is not found")
}

func TestCreate_TheEleventhLiveBot_IsConflict(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	for i := range MaxLive {
		mustCreate(t, s, fmt.Sprintf("bot %d", i))
	}
	_, err := s.Create(as(editor), "c-eng", "one more", "")
	require.ErrorIs(t, err, apperrs.ErrConflict)

	_, err = s.Delete(as(everyone), repo.order[0])
	require.NoError(t, err)
	mustCreate(t, s, "one more")
}

func TestCreate_ANameTakenInAnyCase_IsConflictUntilItsBotIsDeleted(t *testing.T) {
	s := newTestService(newFakeRepo())
	first := mustCreate(t, s, "Café")
	_, err := s.Create(as(editor), "c-eng", " café ", "")
	require.ErrorIs(t, err, apperrs.ErrConflict)

	_, err = s.Delete(as(everyone), first.ID)
	require.NoError(t, err)
	again := mustCreate(t, s, "café")
	assert.Equal(t, "café", again.Name)
}

func TestCreate_InvalidNameOrAvatar_IsInvalid(t *testing.T) {
	big := "data:image/png;base64," + strings.Repeat("A", (maxAvatarBytes/3+1)*4)
	tests := []struct {
		name, botName, avatar string
	}{
		{"empty name", "  ", ""},
		{"name over 80 characters", strings.Repeat("é", MaxNameLength+1), ""},
		{"avatar not a data URI", "CI", "https://example.com/ci.png"},
		{"avatar not an image", "CI", "data:text/plain;base64,aGk="},
		{"avatar not base64", "CI", "data:image/png,hi"},
		{"avatar with a broken payload", "CI", "data:image/png;base64,***"},
		{"avatar over 10MB", "CI", big},
	}
	s := newTestService(newFakeRepo())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Create(as(editor), "c-eng", tt.botName, tt.avatar)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}

	b, err := s.Create(as(editor), "c-eng", strings.Repeat("é", MaxNameLength), "data:image/png;base64,iVBORw0KGgo=")
	require.NoError(t, err, "80 characters is the limit, however many bytes they take")
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", b.Avatar)
}

func TestCreate_IssuesA43CharacterURLSafeToken(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	assert.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`), tokenOf(t, repo, b.ID))
	assert.NotEqual(t, tokenOf(t, repo, b.ID), tokenOf(t, repo, mustCreate(t, s, "CD").ID))
	assert.Equal(t, editor, b.CreatedBy)
}

func TestUpdate_Regenerate_IssuesAFreshToken(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	old := tokenOf(t, repo, b.ID)

	got, err := s.Update(as(editor), b.ID, Changes{Regenerate: true})
	require.NoError(t, err)
	assert.NotEqual(t, old, tokenOf(t, repo, b.ID))
	assert.NotContains(t, got.URL, old)
	last := repo.events[len(repo.events)-1]
	assert.Equal(t, TopicUpdated, last.Topic)
	assert.Equal(t, []Change{ChangeRegenerated}, last.Payload.(Event).Changes)

	_, err = s.Update(as(reader), b.ID, Changes{Regenerate: true})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestUpdate_RenameAndAvatar_PublishWhatChangedAndANoOpNothing(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	mustCreate(t, s, "CD")
	name, avatar := "Builds", "data:image/png;base64,iVBORw0KGgo="

	got, err := s.Update(as(editor), b.ID, Changes{Name: &name, Avatar: &avatar})
	require.NoError(t, err)
	assert.Equal(t, "Builds", got.Name)
	assert.Equal(t, avatar, got.Avatar)
	assert.Equal(t, fixedNow, got.UpdatedAt)
	last := repo.events[len(repo.events)-1].Payload.(Event)
	assert.Equal(t, []Change{ChangeRenamed, ChangeAvatar}, last.Changes)
	assert.Equal(t, "Builds", last.Name)

	events := len(repo.events)
	_, err = s.Update(as(editor), b.ID, Changes{Name: &name, Avatar: &avatar})
	require.NoError(t, err)
	assert.Len(t, repo.events, events, "setting what is already there publishes nothing")

	none, taken := "", "cd"
	got, err = s.Update(as(editor), b.ID, Changes{Avatar: &none})
	require.NoError(t, err)
	assert.Empty(t, got.Avatar, "an empty avatar goes back to the built-in glyph")
	_, err = s.Update(as(editor), b.ID, Changes{Name: &taken})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDelete_WriteWithoutDelete_IsForbidden(t *testing.T) {
	s := newTestService(newFakeRepo())
	b := mustCreate(t, s, "CI")
	_, err := s.Delete(as(editor), b.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	deleted := true
	_, err = s.Update(as(editor), b.ID, Changes{Deleted: &deleted})
	require.ErrorIs(t, err, apperrs.ErrForbidden, "deleting through an update still takes botwebhook:delete")
}

func TestDelete_KillsTheTokenAndIsANoOpTheSecondTime(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	old := tokenOf(t, repo, b.ID)

	got, err := s.Delete(as(deleter), b.ID)
	require.NoError(t, err)
	assert.Empty(t, got.URL)
	assert.Equal(t, deleter, got.DeletedBy)
	assert.Equal(t, fixedNow, *got.DeletedAt)
	assert.NotEqual(t, old, tokenOf(t, repo, b.ID), "the URL it had stops working even where deletion is not checked")
	events := len(repo.events)

	_, err = s.Delete(as(deleter), b.ID)
	require.NoError(t, err)
	assert.Len(t, repo.events, events)
	name := "Builds"
	_, err = s.Update(as(editor), b.ID, Changes{Name: &name})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a deleted bot is restored before it changes")
}

func TestUpdate_Restore_IssuesAFreshTokenAndPublishesRestored(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	issued := tokenOf(t, repo, b.ID)
	_, err := s.Delete(as(everyone), b.ID)
	require.NoError(t, err)
	dead := tokenOf(t, repo, b.ID)
	restore, name := false, "Builds"

	got, err := s.Update(as(editor), b.ID, Changes{Deleted: &restore, Name: &name})
	require.NoError(t, err)
	assert.Nil(t, got.DeletedAt)
	assert.Empty(t, got.DeletedBy)
	assert.NotContains(t, []string{issued, dead}, tokenOf(t, repo, b.ID), "a leaked URL never revives")
	assert.Contains(t, got.URL, tokenOf(t, repo, b.ID))
	assert.Equal(t, []string{TopicCreated, TopicDeleted, TopicRestored, TopicUpdated}, topics(repo.events))
	assert.Equal(t, []Change{ChangeRenamed}, repo.events[3].Payload.(Event).Changes, "a restore's own new token is not a regenerate")
	assert.Equal(t, "Builds", repo.events[2].Payload.(Event).Name)
}

func TestUpdate_RestoreIntoAFullConversationOrATakenName_IsConflict(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")
	_, err := s.Delete(as(everyone), b.ID)
	require.NoError(t, err)
	mustCreate(t, s, "ci")
	restore := false

	_, err = s.Update(as(editor), b.ID, Changes{Deleted: &restore})
	require.ErrorIs(t, err, apperrs.ErrConflict, "its name went to another bot")
	for i := 1; i < MaxLive; i++ {
		mustCreate(t, s, fmt.Sprintf("bot %d", i))
	}
	other := "spare"
	_, err = s.Update(as(editor), b.ID, Changes{Deleted: &restore, Name: &other})
	require.ErrorIs(t, err, apperrs.ErrConflict, "the conversation is full")
}

func TestUpdate_DeleteWithOtherChanges_IsInvalid(t *testing.T) {
	s := newTestService(newFakeRepo())
	b := mustCreate(t, s, "CI")
	deleted, name := true, "Builds"
	_, err := s.Update(as(everyone), b.ID, Changes{Deleted: &deleted, Name: &name})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	got, err := s.Update(as(everyone), b.ID, Changes{Deleted: &deleted})
	require.NoError(t, err)
	assert.NotNil(t, got.DeletedAt)
}

// TestEvents_NeverCarryTheTokenOrTheURL walks a bot through every change and checks no payload names a token it held.
func TestEvents_NeverCarryTheTokenOrTheURL(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b, err := s.Create(as(everyone), "c-dm", "CI", "")
	require.NoError(t, err)
	tokens := []string{tokenOf(t, repo, b.ID)}
	name, restore := "Builds", false
	steps := []func() error{
		func() error {
			_, err := s.Update(as(everyone), b.ID, Changes{Name: &name, Regenerate: true})
			return err
		},
		func() error { _, err := s.Delete(as(everyone), b.ID); return err },
		func() error { _, err := s.Update(as(everyone), b.ID, Changes{Deleted: &restore}); return err },
	}
	for _, step := range steps {
		require.NoError(t, step())
		tokens = append(tokens, tokenOf(t, repo, b.ID))
	}

	require.Len(t, repo.events, 4)
	for _, e := range repo.events {
		raw, err := json.Marshal(e.Payload)
		require.NoError(t, err)
		for _, tok := range tokens {
			assert.NotContains(t, string(raw), tok, e.Topic)
		}
		assert.NotContains(t, string(raw), "/api/botwebhooks/", e.Topic)
		p := e.Payload.(Event)
		assert.Equal(t, Event{BotwebhookID: b.ID, ConversationID: "c-dm", WorkspaceID: "w-acme", Name: p.Name, ActorID: everyone, Changes: p.Changes, MembersOnly: true}, p, e.Topic)
	}
}

func TestService_FailuresBelowTheUseCase_AreReturned(t *testing.T) {
	boom := errors.New("disk full")
	tests := []struct {
		name  string
		setup func(*fakeRepo, *Service)
		call  func(*Service, string) error
	}{
		{"list reads the store", func(r *fakeRepo, _ *Service) { r.listErr = boom }, func(s *Service, _ string) error {
			_, err := s.List(as(editor), "c-eng", false)
			return err
		}},
		{"create checks the conversation's bots", func(r *fakeRepo, _ *Service) { r.listErr = boom }, func(s *Service, _ string) error {
			_, err := s.Create(as(editor), "c-eng", "CD", "")
			return err
		}},
		{"create writes", func(r *fakeRepo, _ *Service) { r.createErr = boom }, func(s *Service, _ string) error {
			_, err := s.Create(as(editor), "c-eng", "CD", "")
			return err
		}},
		{"update reads the bot", func(r *fakeRepo, _ *Service) { r.getErr = boom }, func(s *Service, id string) error {
			_, err := s.Update(as(editor), id, Changes{Regenerate: true})
			return err
		}},
		{"update writes", func(r *fakeRepo, _ *Service) { r.updateErr = boom }, func(s *Service, id string) error {
			_, err := s.Update(as(editor), id, Changes{Regenerate: true})
			return err
		}},
		{"delete writes", func(r *fakeRepo, _ *Service) { r.updateErr = boom }, func(s *Service, id string) error {
			_, err := s.Delete(as(deleter), id)
			return err
		}},
		{"the URL needs the instance address", func(_ *fakeRepo, s *Service) { s.instance = fakeInstance{err: boom} }, func(s *Service, _ string) error {
			_, err := s.List(as(editor), "c-eng", false)
			return err
		}},
		{"the gate fails", func(_ *fakeRepo, s *Service) { s.gate = fakeGate{err: boom} }, func(s *Service, _ string) error {
			_, err := s.List(as(editor), "c-eng", false)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := newTestService(repo)
			b := mustCreate(t, s, "CI")
			tt.setup(repo, s)
			require.ErrorIs(t, tt.call(s, b.ID), boom)
		})
	}

	s := newTestService(newFakeRepo())
	_, err := s.List(as(editor), " ", false)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.Delete(as(editor), "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.Delete(as(editor), "b-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
