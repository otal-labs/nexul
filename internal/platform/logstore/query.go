// Package logstore reads Nexul's own logs back out of OpenObserve on the caller's behalf (ADR 0155). OpenObserve's
// open-source build has no read-only user, so the SQL it runs is built here, from a column list the stream's schema
// supplies, with every value escaped.
package logstore

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

// The limits Nexul puts on a read; OpenObserve itself caps neither the range nor the rows.
const (
	MaxRange     = 7 * 24 * time.Hour
	DefaultRange = time.Hour
	DefaultLimit = 100
	MaxLimit     = 500
	maxValueLen  = 200
)

var sources = []string{"server", "web", "runner", "automations"}

var levels = []string{"debug", "info", "warn", "error"}

// Params are the filters as a caller sent them, the same over MCP and HTTP.
type Params struct {
	Source  string `json:"source,omitempty" jsonschema:"Only lines from this source: server, web, runner or automations. Omit for all of them."`
	Level   string `json:"level,omitempty" jsonschema:"The lowest level to return: debug, info, warn or error. Defaults to info."`
	Since   string `json:"since,omitempty" jsonschema:"How far back to read, as a duration such as 30m or 2d, or a time such as 2026-09-25T10:00:00Z. Defaults to the last hour; at most 7 days before until."`
	Until   string `json:"until,omitempty" jsonschema:"The newest time to read, as a time such as 2026-09-25T10:30:00Z. Defaults to now on the first page; pass the until a page returned back for the next one."`
	TraceID string `json:"trace_id,omitempty" jsonschema:"Only lines of this request, from a trace_id another line carries."`
	Name    string `json:"name,omitempty" jsonschema:"Only lines from the runner or automations host with this name."`
	Text    string `json:"text,omitempty" jsonschema:"Only lines whose message contains this text, ignoring case. It cannot hold a backslash."`
	Limit   int    `json:"limit,omitzero" jsonschema:"How many lines to return, 1 to 500. Defaults to 100."`
	Offset  int    `json:"offset,omitzero" jsonschema:"How many lines to skip, from next_offset of the previous page. Defaults to 0."`
}

// ParamsOf reads Params from an HTTP query string.
func ParamsOf(v url.Values) (Params, error) {
	p := Params{Source: v.Get("source"), Level: v.Get("level"), Since: v.Get("since"), Until: v.Get("until"),
		TraceID: v.Get("trace_id"), Name: v.Get("name"), Text: v.Get("text")}
	var err error
	if p.Limit, err = wholeNumber(v, "limit"); err != nil {
		return Params{}, err
	}
	if p.Offset, err = wholeNumber(v, "offset"); err != nil {
		return Params{}, err
	}
	return p, nil
}

func wholeNumber(v url.Values, key string) (int, error) {
	raw := v.Get(key)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be a whole number", apperrs.ErrInvalid, key)
	}
	return n, nil
}

// Query is Params resolved against a clock and checked against Nexul's limits.
type Query struct {
	Source, Level, TraceID, Name, Text string
	Since, Until                       time.Time
	Window                             paging.Window
}

// Query validates p; an until left out becomes now, which is what pins a paged read.
func (p Params) Query(now time.Time) (Query, error) {
	q := Query{Source: p.Source, Level: strings.ToLower(p.Level), TraceID: p.TraceID, Name: p.Name, Text: p.Text, Until: now.UTC().Truncate(time.Microsecond)}
	if q.Level == "" {
		q.Level = "info"
	}
	if q.Source != "" && !slices.Contains(sources, q.Source) {
		return Query{}, fmt.Errorf("%w: source must be one of %s", apperrs.ErrInvalid, strings.Join(sources, ", "))
	}
	if !slices.Contains(levels, q.Level) {
		return Query{}, fmt.Errorf("%w: level must be one of %s", apperrs.ErrInvalid, strings.Join(levels, ", "))
	}
	for name, value := range map[string]string{"trace_id": p.TraceID, "name": p.Name, "text": p.Text} {
		if err := checkValue(name, value); err != nil {
			return Query{}, err
		}
	}
	if p.Until != "" {
		at, err := parseMoment("until", p.Until, q.Until)
		if err != nil {
			return Query{}, err
		}
		q.Until = at
	}
	q.Since = q.Until.Add(-DefaultRange)
	if p.Since != "" {
		at, err := parseMoment("since", p.Since, q.Until)
		if err != nil {
			return Query{}, err
		}
		q.Since = at
	}
	if !q.Since.Before(q.Until) || q.Until.Sub(q.Since) > MaxRange {
		return Query{}, fmt.Errorf("%w: since must be before until and at most 7 days earlier", apperrs.ErrInvalid)
	}
	q.Window = paging.Window{Offset: max(p.Offset, 0), Limit: p.Limit}
	if q.Window.Limit < 1 {
		q.Window.Limit = DefaultLimit
	}
	q.Window.Limit = min(q.Window.Limit, MaxLimit)
	return q, nil
}

// checkValue refuses what could end the string a value is written into: OpenObserve reads a backslash as an escape,
// so doubling quotes alone is not enough while one can get through.
func checkValue(name, v string) error {
	if len(v) > maxValueLen {
		return fmt.Errorf("%w: %s is at most %d characters", apperrs.ErrInvalid, name, maxValueLen)
	}
	for _, r := range v {
		if r == '\\' || r < ' ' || r == 0x7f {
			return fmt.Errorf("%w: %s cannot hold a backslash or a control character", apperrs.ErrInvalid, name)
		}
	}
	return nil
}

// parseMoment reads a time, or a duration back from ref.
func parseMoment(name, raw string, ref time.Time) (time.Time, error) {
	if at, err := time.Parse(time.RFC3339, raw); err == nil {
		return at.UTC(), nil
	}
	d, err := parseDuration(raw)
	if err != nil || d <= 0 {
		return time.Time{}, fmt.Errorf("%w: %s must be a time such as 2026-09-25T10:00:00Z or a duration such as 30m", apperrs.ErrInvalid, name)
	}
	return ref.Add(-d), nil
}

// parseDuration is time.ParseDuration plus whole days, as in 2d.
func parseDuration(raw string) (time.Duration, error) {
	days, ok := strings.CutSuffix(raw, "d")
	if !ok {
		return time.ParseDuration(raw)
	}
	n, err := strconv.Atoi(days)
	if err != nil || n < 0 || n > 365 {
		return 0, fmt.Errorf("bad days %q", raw)
	}
	return time.Duration(n) * 24 * time.Hour, nil
}
