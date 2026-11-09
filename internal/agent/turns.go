package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// activeTurn is an in-flight turn: the client it runs on and the target to interrupt.
type activeTurn struct {
	client harness.Client
	target harness.Target
	// asked are the request ids of the questions this turn raised, under Service.mu.
	asked []string
	// answered wakes the turn's silence window once an answer to its own question has reached the harness.
	answered chan struct{}
	// requestMessageID is the chat message the turn answers; queued is whether the harness holds the turn in its
	// queue, under Service.mu.
	requestMessageID string
	queued           bool
	// queue is whether the person asked the turn to wait its place rather than steer into the turn under way.
	queue bool
	// ticketID is the ticket a ticket thread belongs to, "" for any other conversation.
	ticketID string
}

// setQueued marks or unmarks the turn's request message as waiting in the harness's queue, once per change; a mention
// sent without Queue is steered into the turn under way instead, and marked only when that is refused (ADR 0172).
func (s *Service) setQueued(ctx context.Context, conversationID string, turn *activeTurn, queued bool) {
	s.mu.Lock()
	changed := turn.queued != queued
	turn.queued = queued
	target := turn.target
	s.mu.Unlock()
	if !changed || turn.requestMessageID == "" {
		return
	}
	if queued && !turn.queue && s.steerQueued(ctx, conversationID, turn, target) {
		return
	}
	frame := StreamFrame{ConversationID: conversationID, QueuedMessageID: turn.requestMessageID, Queued: queued}
	if err := s.live.Publish(ctx, TopicAgentStream, frame); err != nil {
		s.log.Warn("agent: publish queued frame failed", "conversation", conversationID, "error", err)
	}
}

// steerWithin bounds the steer a queued mention gets, which holds up the turn's own updates while it runs.
const steerWithin = 30 * time.Second

// steerQueued hands a queued mention to the conversation's turn under way, whose reply then answers it; never into a run
// of the harness's own, whose reply nobody would post. False leaves it queued.
func (s *Service) steerQueued(ctx context.Context, conversationID string, turn *activeTurn, target harness.Target) bool {
	if !s.underWayBeside(conversationID, turn) {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, steerWithin)
	defer cancel()
	err := turn.client.Steer(ctx, target)
	if err == nil {
		return true
	}
	if !errors.Is(err, apperrs.ErrConflict) {
		s.log.Warn("agent: steer queued mention failed", "conversation", conversationID, "error", err)
	}
	return false
}

// underWayBeside is whether a turn other than turn runs on the conversation outside the harness's queue.
func (s *Service) underWayBeside(conversationID string, turn *activeTurn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.active[conversationID], func(t *activeTurn) bool { return t != turn && !t.queued })
}

func (s *Service) setActive(conversationID string, t *activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active[conversationID] = append(s.active[conversationID], t)
}

// clearActive drops only t, so a turn ending first never orphans another turn's Stop or Answer.
func (s *Service) clearActive(conversationID string, t *activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := slices.DeleteFunc(s.active[conversationID], func(a *activeTurn) bool { return a == t })
	if len(turns) == 0 {
		delete(s.active, conversationID)
		return
	}
	s.active[conversationID] = turns
}

// ask records that t raised requestID, before anyone can see the question to answer it.
func (s *Service) ask(t *activeTurn, requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.asked = append(t.asked, requestID)
}

// answerers splits the conversation's in-flight turns, newest first, into those that asked requestID and the rest.
func (s *Service) answerers(conversationID, requestID string) (asking, others []activeTurn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range slices.Backward(s.active[conversationID]) {
		if slices.Contains(t.asked, requestID) {
			asking = append(asking, *t)
			continue
		}
		others = append(others, *t)
	}
	return asking, others
}

// liveTurns copies the conversation's in-flight turns, newest first.
func (s *Service) liveTurns(conversationID string) []activeTurn {
	s.mu.Lock()
	defer s.mu.Unlock()
	turns := make([]activeTurn, 0, len(s.active[conversationID]))
	for _, t := range slices.Backward(s.active[conversationID]) {
		turns = append(turns, *t)
	}
	return turns
}

// questionFence marks an Agent message whose body is a Question as JSON; the web renders it as the card.
const questionFence = "```nexul-question\n"

// QuestionMessageBody renders q as the Agent message the thread shows as a question card.
func QuestionMessageBody(q harness.Question) string {
	b, err := json.Marshal(q)
	if err != nil {
		return "The Agent asked a question that could not be rendered."
	}
	return questionFence + string(b) + "\n```"
}

// secretFence marks an Agent message whose body is a SecretRequest as JSON; the web renders it as the secret card.
const secretFence = "```nexul-secret\n"

// SecretMessageBody renders r as the Agent message the thread shows as a secret card.
func SecretMessageBody(r harness.SecretRequest) string {
	b, err := json.Marshal(r)
	if err != nil {
		return "The Agent asked for a secret that could not be rendered."
	}
	return secretFence + string(b) + "\n```"
}

// AnswerSecret hands secret to the live turn that asked for it, else another live turn, newest first; an empty
// secret declines. The value goes only to the harness: the thread gets userID's answer without it.
func (s *Service) AnswerSecret(ctx context.Context, conversationID, userID, requestID, secret string) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("%w: request id is required", apperrs.ErrInvalid)
	}
	if _, err := s.conversations.GetConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("answer secret request: %w", err)
	}
	asking, others := s.answerers(conversationID, requestID)
	err := fmt.Errorf("%w: no agent turn on conversation %s is waiting for a secret", apperrs.ErrNotFound, conversationID)
	for _, t := range append(asking, others...) {
		answerer, ok := t.client.(harness.SecretAnswerer)
		if !ok {
			continue
		}
		if err = answerer.AnswerSecret(ctx, t.target, requestID, secret); err == nil || errors.Is(err, apperrs.ErrConflict) {
			break
		}
	}
	if err != nil {
		return err
	}
	wake(asking)
	body := "Answered: saved the secret"
	if secret == "" {
		body = "Answered: declined the secret"
	}
	if err := s.conversations.PostUserMessage(ctx, conversationID, userID, body); err != nil {
		return fmt.Errorf("post secret answer: %w", err)
	}
	return nil
}

// Answer resolves a question through the turn that asked it, else another live turn, newest first; ErrNotFound for none.
func (s *Service) Answer(ctx context.Context, conversationID, requestID string, answer harness.QuestionAnswer) error {
	asking, others := s.answerers(conversationID, requestID)
	if len(asking)+len(others) == 0 {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	var first error
	for _, t := range append(asking, others...) {
		err := t.client.Answer(ctx, t.target, requestID, answer)
		if errors.Is(err, apperrs.ErrConflict) {
			return err
		}
		if err == nil {
			wake(asking)
			return nil
		}
		if first == nil {
			first = err
		}
	}
	return first
}

// wake restarts the silence windows of the turns that asked; any other turn stays paused on its own question.
func wake(turns []activeTurn) {
	for _, t := range turns {
		// A wake already pending covers this answer too.
		select {
		case t.answered <- struct{}{}:
		default:
		}
	}
}

// AnswerFromChat posts the answer as userID's message and resolves the live turn; a turn the harness already closed
// (or one lost to a restart) is resumed as a fresh turn on the same session with the answer as its request.
func (s *Service) AnswerFromChat(ctx context.Context, conversationID, userID, requestID string, answer harness.QuestionAnswer) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("%w: request id is required", apperrs.ErrInvalid)
	}
	if len(answer.Answers) == 0 {
		return fmt.Errorf("%w: an answer is required", apperrs.ErrInvalid)
	}
	body := answer.Summary(nil)
	// The answer is posted only once a turn has it, so a refused answer leaves no message behind to retry over.
	err := s.Answer(ctx, conversationID, requestID, answer)
	gone := errors.Is(err, apperrs.ErrNotFound)
	if err != nil && !gone && !errors.Is(err, apperrs.ErrConflict) {
		return err
	}
	if err := s.conversations.PostUserMessage(ctx, conversationID, userID, body); err != nil {
		return fmt.Errorf("post answer: %w", err)
	}
	if !gone {
		return nil
	}
	go s.runChatTurn(TurnRequest{ConversationID: conversationID, ViaUserID: userID, RequestBody: body,
		Answer: &harness.PendingAnswer{RequestID: requestID, Answer: answer}})
	return nil
}

// Steer hands the queued turn answering messageID to the turn under way on the conversation, whose reply then answers
// it; the caller must be able to read the conversation.
func (s *Service) Steer(ctx context.Context, conversationID, messageID string) error {
	if _, err := s.conversations.GetConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("steer agent turn: %w", err)
	}
	for _, t := range s.liveTurns(conversationID) {
		if t.queued && t.requestMessageID == messageID {
			return t.client.Steer(ctx, t.target)
		}
	}
	return fmt.Errorf("%w: no queued agent turn answers message %s", apperrs.ErrNotFound, messageID)
}

// Interrupt stops every in-flight turn on a conversation (stop control); the caller must be able to read it.
func (s *Service) Interrupt(ctx context.Context, conversationID string) error {
	if _, err := s.conversations.GetConversation(ctx, conversationID); err != nil {
		return fmt.Errorf("interrupt agent turn: %w", err)
	}
	turns := s.liveTurns(conversationID)
	if len(turns) == 0 {
		return fmt.Errorf("%w: no active agent turn on conversation %s", apperrs.ErrNotFound, conversationID)
	}
	var first error
	stopped := false
	for _, t := range turns {
		err := t.client.Interrupt(ctx, t.target)
		stopped = stopped || err == nil
		if first == nil {
			first = err
		}
	}
	if stopped {
		return nil
	}
	return first
}
