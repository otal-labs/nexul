package agent

import (
	"context"
	"errors"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// lookupWithin bounds reading the conversation a session update is about, which runs on the harness's own stream.
const lookupWithin = 10 * time.Second

// ThreadFollower takes over following a thread that has news when it keeps its own record of the work there (a play's
// trail); ok false leaves it to a plain turn, and done closes once its turn ends.
type ThreadFollower interface {
	FollowThread(ctx context.Context, conversationID, threadID, userID, computerID, since string) (done <-chan struct{}, ok bool)
}

// SetFollower registers the follower asked first; call it before any session update can arrive.
func (s *Service) SetFollower(f ThreadFollower) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.follower = f
}

// OnSessionUpdate follows a conversation's harness thread again once the harness reports news on it that no turn here
// is watching: work under way, or a turn newer than the last one a turn here saw end (ADR 0127). It never blocks the
// harness's stream for longer than one lookup.
func (s *Service) OnSessionUpdate(userID, computerID string, u harness.SessionUpdate) {
	if u.Gone || u.SessionID == "" || userID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), lookupWithin)
	defer cancel()
	conv, err := s.conversations.ConversationByThread(ctx, u.SessionID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return
	}
	if err != nil {
		s.log.Warn("agent: read the conversation of a harness thread failed", "thread", u.SessionID, "error", err)
		return
	}
	if !s.claimFollow(conv.ID) {
		return
	}
	since, news := conv.SeenMarker, u.Working || u.Latest != conv.SeenMarker
	// A thread from before Nexul kept a marker starts from what is there now, so its past never replays.
	if since == "" {
		s.markSeen(ctx, conv.ID, &harness.TurnResult{Marker: u.Latest})
		news = u.Working
	}
	if !news {
		s.releaseFollow(conv.ID)
		return
	}
	go s.follow(conv.ID, u.SessionID, userID, computerID, since)
}

// follow runs the catch-up through the follower when it takes it, else as a plain turn, and frees the conversation after.
func (s *Service) follow(conversationID, threadID, userID, computerID, since string) {
	defer s.releaseFollow(conversationID)
	ctx := context.Background()
	s.mu.Lock()
	follower := s.follower
	s.mu.Unlock()
	if follower != nil {
		if done, ok := follower.FollowThread(ctx, conversationID, threadID, userID, computerID, since); ok {
			<-done
			return
		}
	}
	s.runChatTurn(TurnRequest{ConversationID: conversationID, ViaUserID: userID, Watch: true, Since: since, Target: &TargetOverride{ComputerID: computerID}})
}

// claimFollow reserves a conversation no turn is running on for one catch-up; false when a turn or a catch-up has it.
func (s *Service) claimFollow(conversationID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.active[conversationID]) > 0 || s.following[conversationID] {
		return false
	}
	s.following[conversationID] = true
	return true
}

func (s *Service) releaseFollow(conversationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.following, conversationID)
}
