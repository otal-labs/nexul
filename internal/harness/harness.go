// Package harness is the seam between Nexul and an agent harness (T3 Code today, others later): one Client per
// kind, registered by Kind, and only harness-neutral types cross it (ADR 0054).
package harness

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Kind names a harness implementation; it is stored on every paired computer and picks the Client from a Registry.
type Kind string

const (
	// KindT3Code is the T3 Code desktop app's server on orchestration protocol 1.
	KindT3Code Kind = "t3code"
	// KindT3CodeV2 is the same server once it runs T3's orchestrator V2, protocol 2.
	KindT3CodeV2 Kind = "t3code-v2"
)

// ErrProtocol marks a harness speaking a protocol Nexul cannot follow; the refusal's text is the message to show.
var ErrProtocol = errors.New("unsupported harness protocol")

// ErrSessionGone marks a session the harness no longer has, deleted there or never kept.
var ErrSessionGone = errors.New("the harness session is gone")

// ProtocolRefusal is an error matching ErrProtocol and ErrInvalid whose text is msg alone, so callers show it as is.
func ProtocolRefusal(msg string) error {
	return protocolRefusal(msg)
}

type protocolRefusal string

func (r protocolRefusal) Error() string { return string(r) }

// Unwrap includes ErrInvalid so the HTTP gateway and MCP show the refusal instead of hiding it as an internal error.
func (protocolRefusal) Unwrap() []error { return []error{ErrProtocol, apperrs.ErrInvalid} }

// MovedError is a client finding its harness moved on to the protocol kind To's client speaks; Forward follows it.
type MovedError struct {
	To Kind
}

func (e *MovedError) Error() string { return fmt.Sprintf("the harness moved on to %s", e.To) }

// Session is what every authenticated call needs to reach one paired computer; BearerToken is plaintext here.
type Session struct {
	// ComputerID names the paired computer, so Forward's moved callback knows which row to switch.
	ComputerID  string
	Name        string
	ServerURL   string
	BearerToken string
}

// PairResult is what a successful pairing hands back for storage.
type PairResult struct {
	BearerToken string
	ExpiresIn   time.Duration
	Version     string
	// Kind is the client the pairing landed on, which a caller stores on the computer.
	Kind Kind
}

// Project is one entry of a computer's project registry, for the settings UI's picker.
type Project struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
}

// ProviderModel is one model a provider instance offers, with the options a turn may set on it.
type ProviderModel struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
	// SubProvider names where a routed model comes from, such as "GitHub Copilot"; empty for the provider's own.
	SubProvider string        `json:"sub_provider,omitempty"`
	IsNew       bool          `json:"is_new,omitempty"`
	IsLegacy    bool          `json:"is_legacy,omitempty"`
	Options     []ModelOption `json:"options,omitempty"`
}

// OptionType is how a model option is set: one choice out of several, or on and off.
type OptionType string

const (
	OptionSelect OptionType = "select"
	OptionSwitch OptionType = "boolean"
)

// ModelOption is one setting the harness offers per model, such as reasoning level, context window, or fast mode.
type ModelOption struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	Description string         `json:"description,omitempty"`
	Type        OptionType     `json:"type"`
	Choices     []OptionChoice `json:"choices,omitempty"`
	// DefaultOn is a switch's value when the turn leaves it unset.
	DefaultOn bool `json:"default_on,omitempty"`
}

// OptionChoice is one choice of a select option.
type OptionChoice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// OptionSetting is one model option set for a turn: a choice id for a select, a bool for a switch.
type OptionSetting struct {
	ID    string `json:"id" jsonschema:"The option's id as the harness lists it, for example effort or fastMode."`
	Value any    `json:"value" jsonschema:"A choice id for a select option, for example high; true or false for a switch."`
}

// CleanOptions trims and checks settings at a trust boundary; an unset option is left to the harness default.
func CleanOptions(settings []OptionSetting) ([]OptionSetting, error) {
	var out []OptionSetting
	seen := map[string]bool{}
	for _, s := range settings {
		id := strings.TrimSpace(s.ID)
		if id == "" {
			return nil, fmt.Errorf("%w: a model option needs its id", apperrs.ErrInvalid)
		}
		if seen[id] {
			return nil, fmt.Errorf("%w: model option %s is set twice", apperrs.ErrInvalid, id)
		}
		seen[id] = true
		switch v := s.Value.(type) {
		case bool:
			out = append(out, OptionSetting{ID: id, Value: v})
		case string:
			if strings.TrimSpace(v) == "" {
				return nil, fmt.Errorf("%w: model option %s needs a value", apperrs.ErrInvalid, id)
			}
			out = append(out, OptionSetting{ID: id, Value: strings.TrimSpace(v)})
		default:
			return nil, fmt.Errorf("%w: model option %s must be a choice id or true/false", apperrs.ErrInvalid, id)
		}
	}
	return out, nil
}

// Provider is a usable provider instance; ID is what a turn routes on, Driver the kind setup is confirmed under.
type Provider struct {
	ID      string          `json:"id"`
	Driver  string          `json:"driver"`
	Name    string          `json:"name"`
	Version string          `json:"-"` // the provider CLI's version, "" when unknown
	Models  []ProviderModel `json:"models"`
}

// Target names the computer, harness-side project, model choice and durable session a turn runs against.
type Target struct {
	Session   Session
	ProjectID string
	Provider  string
	Model     string
	// ModelOptions ride with Model; a reused protocol-2 thread switches to them when they differ, a protocol-1 session keeps its own.
	ModelOptions []OptionSetting
	// SessionID is the harness-side thread; empty means create one.
	SessionID string
	// TurnID is the StartResult.TurnID of the turn Interrupt stops; empty means whatever runs on SessionID.
	TurnID string
	// Worktree creates a new thread in a fresh git worktree of the project rather than in its folder.
	Worktree bool
	// Since is a TurnResult.Marker: Watch follows from the first turn after it, so a catch-up misses none; empty follows the newest.
	Since string
	// KeepSession makes StartTurn fail with ErrSessionGone instead of starting a new session when SessionID is gone.
	KeepSession bool
}

// Conn is a held connection; Done fires when the server drops it.
type Conn interface {
	Done() <-chan struct{}
	Close() error
}

// SessionUpdate is a change to one session on a held computer, as the harness's own apps see it.
type SessionUpdate struct {
	SessionID string
	// Latest is the session's newest turn, in TurnResult.Marker's terms.
	Latest string
	// Working is whether a turn on the session is under way.
	Working bool
	// Gone is a session deleted in the harness.
	Gone bool
}

// SessionWatcher is a Client whose held connection also reports session changes; a Client that is not one only holds.
type SessionWatcher interface {
	// WatchSessions holds like Hold and calls onUpdate for every session as held, then for each change, until Done.
	WatchSessions(ctx context.Context, s Session, onUpdate func(SessionUpdate)) (Conn, error)
}

// Snapshot is one cumulative reply, re-emitted per MessageID until Streaming goes false.
type Snapshot struct {
	MessageID string
	Text      string
	Streaming bool
}

// ActivityKind classifies one step of a turn; harness-neutral so the trail and the thread render every harness alike.
type ActivityKind string

const (
	ActivityToolCall   ActivityKind = "tool_call"
	ActivityToolResult ActivityKind = "tool_result"
	ActivityText       ActivityKind = "text"
	ActivityQuestion   ActivityKind = "question"
	ActivityOther      ActivityKind = "other"
	// ActivityNote is a line about the run itself rather than a step of the agent's, shown muted.
	ActivityNote ActivityKind = "note"
	// ActivityUserMessage is a message the person wrote in the harness itself; Detail is its text, Tool names the harness.
	ActivityUserMessage ActivityKind = "user_message"
)

// MaxActivityDetail caps Activity.Detail: a tool result can carry a whole file, and a trail keeps hundreds of steps.
const MaxActivityDetail = 8 << 10

// Activity is one step the harness reported mid-turn: a tool call and its result, a question, or assistant text.
type Activity struct {
	Kind ActivityKind
	// CallID groups the updates of one tool call so a consumer replaces the step instead of appending a duplicate.
	CallID string
	Tool   string
	// Summary is the one-line label: the argument preview for a tool, the first line for text, " · failed" appended on failure.
	Summary string
	// Detail is the raw arguments and result as JSON, or the full text, cut at MaxActivityDetail bytes.
	Detail string
	At     time.Time
}

// CapDetail cuts s at MaxActivityDetail bytes on a rune boundary.
func CapDetail(s string) string {
	return CapBytes(s, MaxActivityDetail)
}

// CapBytes cuts s at n bytes on a rune boundary.
func CapBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// Preview flattens s to one line for a row label, cut to n runes and an ellipsis when it is longer.
func Preview(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// Approval surfaces a stray prompt the harness already auto-declined, for the pipeline's system message.
type Approval struct {
	Kind    string
	Summary string
}

// QuestionOption is one choice of a question; Value is what the harness wants back, the label when empty.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
}

// QuestionItem is one question of a request; a harness may ask several in one go.
type QuestionItem struct {
	ID          string           `json:"id"`
	Text        string           `json:"text"`
	Header      string           `json:"header,omitempty"`
	Options     []QuestionOption `json:"options"`
	MultiSelect bool             `json:"multi_select,omitempty"`
}

// Question is a harness turn stopping to ask the user; the turn stays open until RequestID is answered.
type Question struct {
	RequestID string         `json:"request_id"`
	Questions []QuestionItem `json:"questions"`
}

// AnswerValue answers one QuestionItem: the chosen option values, or free text when Text is set.
type AnswerValue struct {
	Selected []string `json:"selected,omitempty"`
	Text     string   `json:"text,omitempty"`
}

// QuestionAnswer answers every QuestionItem of one Question, keyed by QuestionItem.ID.
type QuestionAnswer struct {
	Answers map[string]AnswerValue `json:"answers"`
}

// Summary is the answer as a thread message: one line for one question, a bullet per question otherwise.
func (a QuestionAnswer) Summary(q *Question) string {
	if q == nil || len(q.Questions) <= 1 {
		for _, v := range a.Answers {
			return "Answered: " + v.String()
		}
		return "Answered."
	}
	lines := []string{"Answered:"}
	for _, item := range q.Questions {
		lines = append(lines, "- "+Preview(item.Text, 80)+": "+a.Answers[item.ID].String())
	}
	return strings.Join(lines, "\n")
}

// PendingAnswer is an answer to a question whose turn already ended; the next turn carries it to the harness.
type PendingAnswer struct {
	RequestID string
	Answer    QuestionAnswer
}

// String renders one answer: the free text, else the chosen values comma-joined.
func (v AnswerValue) String() string {
	if v.Text != "" {
		return v.Text
	}
	return strings.Join(v.Selected, ", ")
}

// TurnState is a turn's terminal outcome.
type TurnState string

const (
	TurnDone        TurnState = "done"
	TurnInterrupted TurnState = "interrupted"
	TurnError       TurnState = "error"
)

// TurnResult is a turn's terminal state.
type TurnResult struct {
	State     TurnState
	LastError string
	// LeftRunning is a done turn that stopped waiting for work it handed off, which still runs in the harness.
	LeftRunning bool
	// Marker names the newest turn this one saw end on the harness, for SessionUpdate.Latest and Target.Since; empty
	// when the harness keeps none or the turn ended without hearing from the harness, such as a lost connection.
	Marker string
	// SessionGone is a turn that never started because the session it had to keep is gone.
	SessionGone bool
}

// The states of a Handoff.
const (
	HandoffRunning     = "running"
	HandoffDone        = "done"
	HandoffFailed      = "failed"
	HandoffInterrupted = "interrupted"
	// HandoffLeftRunning is work still running when the turn stopped waiting for it.
	HandoffLeftRunning = "left_running"
)

// Handoff is one cumulative snapshot of work a turn handed to another agent, re-emitted on change and keyed by ID.
type Handoff struct {
	ID     string
	Driver string
	Model  string
	Title  string
	Prompt string
	State  string
	Reply  string
	Steps  []Activity
}

// Update is one item off a turn's stream, exactly one field set; a Terminal update is always last.
// A Question is not terminal: the turn stays open on the harness until Client.Answer or Interrupt.
type Update struct {
	Snapshot *Snapshot
	Activity *Activity
	Approval *Approval
	Question *Question
	Handoff  *Handoff
	Terminal *TurnResult
}

// Attachment is one file a turn hands to the harness alongside its prompt; harness-neutral, bytes inline.
type Attachment struct {
	Name  string
	MIME  string
	Bytes []byte
}

// TurnPrompts carries both prompt shapes; the harness picks Full (new session) or Incremental (reused).
type TurnPrompts struct {
	Full        string
	Incremental string
	// Attachments belong to Full: they are the images its ticket, doc, and memory bodies refer to.
	Attachments []Attachment
	// Answer resolves the question still open on a reused session before the prompt is sent.
	Answer *PendingAnswer
}

// StartResult is what starting a turn hands back: the session id actually used, plus the update stream.
type StartResult struct {
	SessionID string
	// TurnID names this turn among others on the same session, for Target.TurnID; empty when the harness needs none.
	TurnID  string
	Updates <-chan Update
	// PromptSent is false when the harness only resolved a pending answer, so the messages since are still unsent.
	PromptSent bool
}

// Client is everything Nexul asks of one harness kind. Harness-specific concepts stay behind it.
type Client interface {
	Kind() Kind
	// Pair trades whatever the user typed for that harness (T3: the `t3 pair` token) for a bearer session and version.
	Pair(ctx context.Context, serverURL, secret string) (PairResult, error)
	// Version is the unauthenticated probe, re-checked at turn start to warn about drift since pairing.
	Version(ctx context.Context, serverURL string) (string, error)
	ListProjects(ctx context.Context, s Session) ([]Project, error)
	ListProviders(ctx context.Context, s Session) ([]Provider, error)
	// Hold keeps a live connection open for presence; a harness with nothing to hold returns a nil Conn.
	Hold(ctx context.Context, s Session) (Conn, error)
	// StartTurn starts one turn; title names a freshly created session, ignored when SessionID is reused.
	StartTurn(ctx context.Context, t Target, title string, prompts TurnPrompts) (StartResult, error)
	// Watch follows the turn on t's session that another watcher started; with nothing running it ends at once, done.
	Watch(ctx context.Context, t Target) (StartResult, error)
	// Interrupt aborts the turn t.TurnID names on t's session, or whatever turn is active there when it names none.
	Interrupt(ctx context.Context, t Target) error
	// Answer resolves the pending Question requestID on t's session so the turn continues; ErrConflict when the
	// harness already holds an answer for it.
	Answer(ctx context.Context, t Target, requestID string, answer QuestionAnswer) error
	// Settle moves t's idle session out of the harness's active list; the harness wakes it on new activity.
	Settle(ctx context.Context, t Target) error
}

// ImageTaker is a Client that sends its agent only some image types; a Client that is not one sends every image.
type ImageTaker interface {
	TakesImage(mime string) bool
}

// Registry maps each supported kind to its client; the composition root builds it once.
type Registry map[Kind]Client
