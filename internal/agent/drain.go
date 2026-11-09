package agent

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/harness"
)

// drainTurn reads a turn's updates to completion, forwarding snapshots, hand-offs and approvals live.
func (s *Service) drainTurn(ctx context.Context, turn *activeTurn, conversationID, viaUserID string, updates <-chan harness.Update, obs Observer, window *silenceWindow) (finalText string, handoffs []harness.Handoff, term *harness.TurnResult) {
	frame := StreamFrame{ConversationID: conversationID, TicketID: turn.ticketID, Streaming: true}
	publish := func() {
		if err := s.live.Publish(ctx, TopicAgentStream, frame); err != nil {
			s.log.Warn("agent: publish stream frame failed", "conversation", conversationID, "error", err)
		}
	}
	var seg textSegments
	for {
		u, end, ok := window.receive(ctx, updates)
		if end != nil {
			return finalText, handoffs, end
		}
		if !ok {
			return finalText, handoffs, term
		}
		switch {
		case u.Snapshot != nil:
			finalText = onSnapshot(*u.Snapshot, finalText, &frame, publish, &seg, obs)
		case u.Queued != nil:
			s.setQueued(ctx, conversationID, turn, *u.Queued)
		case u.Activity != nil:
			s.onActivity(ctx, conversationID, viaUserID, *u.Activity, &frame, publish, &seg, obs)
		case u.Question != nil, u.Secret != nil:
			s.postRequest(ctx, turn, conversationID, viaUserID, u, obs)
		case u.Answered != nil:
			obs.OnAnswered(*u.Answered)
		case u.Handoff != nil:
			handoffs = keepHandoff(handoffs, *u.Handoff)
			h := chat.NewHandoff(*u.Handoff)
			frame.Handoff = &h
			publish()
			frame.Handoff = nil
			obs.OnSnapshot()
		case u.Approval != nil:
			s.postSystemMessage(ctx, conversationID, viaUserID, fmt.Sprintf(
				"Agent's environment asked for approval (%s: %s) — auto-declined; adjust its runtime mode if you want it to proceed unattended.",
				u.Approval.Kind, u.Approval.Summary))
		case u.Terminal != nil:
			term = u.Terminal
		}
	}
}

// postRequest lands a question or secret request as the Agent's own message so the thread shows its card; the stream
// bubble yields to it.
func (s *Service) postRequest(ctx context.Context, turn *activeTurn, conversationID, viaUserID string, u harness.Update, obs Observer) {
	if u.Secret != nil {
		s.ask(turn, u.Secret.ID)
		if _, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, SecretMessageBody(*u.Secret), nil); err != nil {
			s.log.Error("agent: post secret request failed", "conversation", conversationID, "error", err)
		}
		return
	}
	s.ask(turn, u.Question.RequestID)
	if _, err := s.conversations.PostAgentReply(ctx, conversationID, viaUserID, QuestionMessageBody(*u.Question), nil); err != nil {
		s.log.Error("agent: post question failed", "conversation", conversationID, "error", err)
	}
	obs.OnQuestion(*u.Question)
}

// onSnapshot streams one reply snapshot into the bubble and returns the reply text so far.
func onSnapshot(snap harness.Snapshot, finalText string, frame *StreamFrame, publish func(), seg *textSegments, obs Observer) string {
	// A streaming:false finalize may carry empty text (a close marker); only non-empty text may set the reply.
	if snap.Text != "" {
		finalText = snap.Text
	}
	// One message finishing is not the turn finishing; the frame streams until finishTurn clears it.
	frame.MessageID, frame.Text = snap.MessageID, snap.Text
	frame.Activity, frame.ActivityKind, frame.ActivityTool, frame.ActivityCallID, frame.ActivityText = "", "", "", "", ""
	publish()
	obs.OnSnapshot()
	seg.observe(snap)
	if !snap.Streaming {
		seg.flush(obs)
	}
	return finalText
}

// onActivity records a step; the Agent's own is the live bubble's step line, while a message the person wrote in the
// harness is theirs and lands in the thread instead. Either closes the text written before it.
func (s *Service) onActivity(ctx context.Context, conversationID, viaUserID string, a harness.Activity, frame *StreamFrame, publish func(), seg *textSegments, obs Observer) {
	if a.Kind == harness.ActivityUserMessage {
		seg.flush(obs)
		s.relayUserMessage(ctx, conversationID, viaUserID, a)
		obs.OnActivity(a)
		return
	}
	frame.Activity, frame.ActivityKind, frame.ActivityTool, frame.ActivityCallID, frame.ActivityText = a.Summary, a.Kind, a.Tool, a.CallID, ""
	if a.Kind == harness.ActivityReasoning {
		frame.ActivityText = a.Detail
	}
	publish()
	if a.Kind == harness.ActivityToolCall || a.Kind == harness.ActivityReasoning {
		seg.flush(obs)
	}
	obs.OnActivity(a)
}

// relayUserMessage posts a message the person wrote in the harness into the thread as theirs; a failure is only logged.
func (s *Service) relayUserMessage(ctx context.Context, conversationID, userID string, a harness.Activity) {
	body := a.Detail
	if body == "" {
		body = a.Summary
	}
	if err := s.conversations.PostHarnessMessage(ctx, conversationID, userID, body, a.Tool, a.CallID, a.At); err != nil {
		s.log.Error("agent: relay a message written in the harness failed", "conversation", conversationID, "error", err)
	}
}

// keepHandoff replaces h's earlier snapshot, else appends it, so the reply keeps the order the hand-offs started in.
func keepHandoff(hs []harness.Handoff, h harness.Handoff) []harness.Handoff {
	if i := slices.IndexFunc(hs, func(x harness.Handoff) bool { return x.ID == h.ID }); i >= 0 {
		hs[i] = h
		return hs
	}
	return append(hs, h)
}

// silenceWindow ends a turn after d without a harness update; a question pauses it until answered, and a zero d never fires.
type silenceWindow struct {
	d        time.Duration
	timer    *time.Timer
	paused   bool
	answered <-chan struct{}
}

func newSilenceWindow(d time.Duration, answered <-chan struct{}) *silenceWindow {
	w := &silenceWindow{d: d, answered: answered}
	if d > 0 {
		w.timer = time.NewTimer(d)
	}
	return w
}

// receive returns the next update and whether the stream is still open, or end when the window or ctx closed the turn first.
func (w *silenceWindow) receive(ctx context.Context, updates <-chan harness.Update) (u harness.Update, end *harness.TurnResult, ok bool) {
	var expired <-chan time.Time
	if w.timer != nil {
		expired = w.timer.C
	}
	for {
		select {
		case u, ok = <-updates:
			w.paused = w.paused && u.Answered == nil
			w.heard(u.Question != nil || u.Secret != nil)
			return u, nil, ok
		case <-w.answered:
			w.paused = false
			w.heard(false)
		case <-expired:
			return u, &harness.TurnResult{State: harness.TurnError, LastError: fmt.Sprintf(
				"turn gave no completion signal after %s of silence — the harness may have finished without our client seeing it; check the server log's harness client warnings", w.d)}, false
		case <-ctx.Done():
			return u, &harness.TurnResult{State: harness.TurnError, LastError: "turn cancelled: " + context.Cause(ctx).Error()}, false
		}
	}
}

// heard restarts the window; a question or secret request stops it, since the user's silence is not the harness's.
func (w *silenceWindow) heard(question bool) {
	if w.timer == nil {
		return
	}
	if question {
		w.paused = true
		w.timer.Stop()
		return
	}
	if w.paused {
		return
	}
	w.timer.Reset(w.d)
}

// textSegments cuts one growing reply into the pieces written between tool calls, so a trail reads what the
// Agent said before each action instead of the whole reply once at the end.
type textSegments struct {
	messageID string
	text      string
	emitted   int
}

func (s *textSegments) observe(snap harness.Snapshot) {
	if snap.MessageID != s.messageID {
		s.messageID, s.emitted = snap.MessageID, 0
	}
	if snap.Text != "" {
		s.text = snap.Text
	}
	if s.emitted > len(s.text) {
		s.emitted = 0
	}
}

// flush emits the text not yet seen as a step and marks it seen; nothing new means no step.
func (s *textSegments) flush(obs Observer) {
	from := s.emitted
	piece := strings.TrimSpace(s.text[from:])
	s.emitted = len(s.text)
	if piece == "" {
		return
	}
	a := textActivity(piece)
	// Named by where it starts in its message, so a catch-up that replays the message replaces the step it already has.
	if s.messageID != "" {
		a.CallID = fmt.Sprintf("text:%s:%d", s.messageID, from)
	}
	obs.OnActivity(a)
}

// textActivity is one piece of the reply as a transcript step.
func textActivity(text string) harness.Activity {
	return harness.Activity{Kind: harness.ActivityText, Summary: harness.Preview(text, 160), Detail: harness.CapDetail(text), At: time.Now().UTC()}
}
