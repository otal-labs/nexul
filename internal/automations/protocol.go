package automations

import (
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// FrameType discriminates the dial-in wire protocol frames (ADR 0046), mirroring internal/runner's protocol shape.
type FrameType string

const (
	// FrameAnnounce is sent right after connecting, carrying the surface SyncFromCode overwrites the record with.
	FrameAnnounce FrameType = "announce"
	// FrameHello answers a successful announce: current config values and
	// workspace secrets, pushed once per connection.
	FrameHello FrameType = "hello"
	// FrameEvent carries one subscribed event down to the automation.
	FrameEvent FrameType = "event"
	// FrameRunStarted/Log/Finished/Crashed report one run back up the
	// connection.
	FrameRunStarted  FrameType = "run_started"
	FrameRunLog      FrameType = "run_log"
	FrameRunFinished FrameType = "run_finished"
	FrameRunCrashed  FrameType = "run_crashed"
)

// Outcomes carried on a run_finished frame; a crash where the worker never reports rides on run_crashed instead.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

// Frame's Validate enforces per-type requirements before dispatch.
type Frame struct {
	Type          FrameType         `json:"type"`
	Name          string            `json:"name,omitempty"`
	Description   string            `json:"description,omitempty"`
	Subscriptions []string          `json:"subscriptions,omitempty"`
	ConfigSchema  json.RawMessage   `json:"config_schema,omitempty"`
	ConfigValues  json.RawMessage   `json:"config_values,omitempty"`
	Secrets       map[string]string `json:"secrets,omitempty"`
	RunID         string            `json:"run_id,omitempty"`
	EventID       string            `json:"event_id,omitempty"`
	Topic         string            `json:"topic,omitempty"`
	Payload       json.RawMessage   `json:"payload,omitempty"`
	Log           string            `json:"log,omitempty"`
	Outcome       string            `json:"outcome,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// Validate checks the fields required by the frame's type. Unknown types and
// malformed values return ErrInvalid wrapped with context.
func (f *Frame) Validate() error {
	if f.Type == "" {
		return fmt.Errorf("%w: frame type is required", apperrs.ErrInvalid)
	}
	switch f.Type {
	case FrameAnnounce:
		if f.Name == "" {
			return fmt.Errorf("%w: announce requires name", apperrs.ErrInvalid)
		}
	case FrameHello:
		// server -> automation; built internally, always valid.
	case FrameEvent:
		if f.RunID == "" || f.EventID == "" || f.Topic == "" {
			return fmt.Errorf("%w: event requires run_id, event_id and topic", apperrs.ErrInvalid)
		}
	case FrameRunStarted, FrameRunLog, FrameRunCrashed:
		if f.RunID == "" {
			return fmt.Errorf("%w: %s requires run_id", apperrs.ErrInvalid, f.Type)
		}
	case FrameRunFinished:
		if f.RunID == "" || !oneOf(f.Outcome, OutcomeSuccess, OutcomeFailure) {
			return fmt.Errorf("%w: run_finished requires run_id and a valid outcome", apperrs.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unknown frame type %q", apperrs.ErrInvalid, f.Type)
	}
	return nil
}

// ParseFrame decodes and validates a frame from its JSON wire form.
func ParseFrame(data []byte) (*Frame, error) {
	var f Frame
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode frame: %w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

// Encode validates and serializes the frame.
func (f *Frame) Encode() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode frame: %w", err)
	}
	return data, nil
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}
