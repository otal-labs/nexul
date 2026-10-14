// Package welcome works out each person's Welcome wizard progress: the steps their roles give them, which are done,
// which they skipped, and whether any lights the account menu's dot.
package welcome

import (
	"time"
)

// State is where one of a person's steps stands.
type State string

const (
	Done    State = "done"
	Skipped State = "skipped"
	Left    State = "left"
)

// Step is one of a person's Welcome wizard steps.
type Step struct {
	Key   string `json:"key" enum:"github,computer,projects,deploy_server,phone"`
	State State  `json:"state" enum:"done,skipped,left"`
	// At is when it was skipped, or for the phone when one first signed in.
	At *time.Time `json:"at,omitempty"`
	// Given is true once the step was given to the person again after their first run; only projects reads it.
	Given bool `json:"given"`
}

// Progress is one person's Welcome wizard: the first run, then each step a role they hold gives them.
type Progress struct {
	FirstRunDone bool   `json:"first_run_done"`
	Steps        []Step `json:"steps"`
	// Dot is whether the account menu shows its dot: a step neither done nor skipped, projects only once given.
	Dot bool `json:"dot"`
	// InAProject holds whether or not projects is a step of theirs, since Team names a person in no project.
	InAProject bool `json:"in_a_project"`
}

// Record is what the welcome domain stores for one person's step; everything else is read from other domains.
type Record struct {
	UserID    string
	Step      string
	SkippedAt *time.Time
	GivenAt   *time.Time
	DoneAt    *time.Time
}

// Facts is what other domains hold for one person, each read through a seam.
type Facts struct {
	Ticked       []string
	GitHub       bool
	Computer     bool
	InAProject   bool
	DeployServer bool
	FirstRunDone bool
}
