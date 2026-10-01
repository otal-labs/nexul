// Package templates implements the instance layer every template resolves through (ADR 0103): the code default,
// then the instance template, then the workspace or project that holds its own text.
package templates

import (
	"context"
	"time"
)

// Scope is the layer a template lives at.
type Scope string

// The layers, top to bottom; a kind lives at the instance and at exactly one layer below it.
const (
	ScopeInstance  Scope = "instance"
	ScopeWorkspace Scope = "workspace"
	ScopeProject   Scope = "project"
)

// Location names one place a template lives: the instance, a workspace, or a project.
type Location struct {
	Scope       Scope  `json:"scope"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
}

// Instance is the instance's own location.
var Instance = Location{Scope: ScopeInstance}

// Template is one template's text at one location.
type Template struct {
	Kind        string `json:"kind"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Scope       Scope  `json:"scope"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Body        string `json:"body"`
	// DefaultBody is what a reset here gives: the code default at the instance, the instance's text below it.
	DefaultBody string `json:"default_body"`
	// Edited is false while this location shows DefaultBody: never edited, following the instance, or reset.
	Edited bool `json:"edited"`
	// Follows says the kind's lower layer follows the instance live until edited, rather than copying it at creation.
	Follows   bool       `json:"follows"`
	UpdatedBy string     `json:"updated_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// Record is a stored instance template; no record means the code default.
type Record struct {
	Kind      string
	Key       string
	Body      string
	UpdatedBy string
	UpdatedAt time.Time
}

// Default is one key's code default and its human name.
type Default struct {
	Key  string
	Name string
	Body string
}

// Layer reads and writes a kind's template below the instance, through the domain that owns it and its permissions.
type Layer interface {
	// Read returns the text at the location and whether the location holds its own rather than following the instance.
	Read(ctx context.Context, at Location, key string) (body string, own bool, err error)
	Write(ctx context.Context, at Location, key, body string) error
	// Reset returns the location to the instance's text: following it again, or a fresh copy of instanceBody.
	Reset(ctx context.Context, at Location, key, instanceBody string) error
}

// Kind registers one kind of template: its keys and code defaults, the layer below the instance, and how it is read.
type Kind struct {
	Name string
	// Below is the scope of the layer under the instance: workspace or project.
	Below Scope
	// Follows is true when the lower layer is stored only once edited, so an unedited one reads the instance live.
	Follows  bool
	Defaults []Default
	// Check validates a body before it is stored at the instance; nil accepts any text.
	Check func(body string) error
	Layer Layer
}
