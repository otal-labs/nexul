// Package labels is the one rule for the labels on a computer and the needs a play names (ADR 0197).
package labels

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Max and maxLength bound one list of labels.
const (
	Max       = 20
	maxLength = 32
)

var pattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Normalize lowercases and trims each label, drops blanks and repeats, and refuses one outside the rule.
func Normalize(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, label := range in {
		label = strings.ToLower(strings.TrimSpace(label))
		if label == "" || slices.Contains(out, label) {
			continue
		}
		if len(label) > maxLength || !pattern.MatchString(label) {
			return nil, fmt.Errorf("%w: label %q: use lowercase letters, digits and dashes, up to %d characters", apperrs.ErrInvalid, label, maxLength)
		}
		out = append(out, label)
	}
	if len(out) > Max {
		return nil, fmt.Errorf("%w: up to %d labels", apperrs.ErrInvalid, Max)
	}
	return out, nil
}

// ComputerRule narrows the computers an auto play's runs may land on: one of LabelsIn when set, none of LabelsOut, and
// one of IDs when set (ADR 0214).
type ComputerRule struct {
	LabelsIn  []string `json:"labels_in,omitempty" jsonschema:"A computer needs at least one of these labels; empty for no demand."`
	LabelsOut []string `json:"labels_out,omitempty" jsonschema:"A computer with any of these labels is left out."`
	IDs       []string `json:"ids,omitempty" jsonschema:"Only these computers, when person is the run_on; empty for any."`
}

// Normalize tidies each list as labels are, and drops repeated or blank ids; the result has no nil list.
func (r ComputerRule) Normalize() (ComputerRule, error) {
	var err error
	if r.LabelsIn, err = Normalize(r.LabelsIn); err != nil {
		return r, err
	}
	if r.LabelsOut, err = Normalize(r.LabelsOut); err != nil {
		return r, err
	}
	ids := make([]string, 0, len(r.IDs))
	for _, id := range r.IDs {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	r.IDs = ids
	return r, nil
}

// Empty is a rule that leaves every computer in.
func (r ComputerRule) Empty() bool {
	return len(r.LabelsIn) == 0 && len(r.LabelsOut) == 0 && len(r.IDs) == 0
}

// LeftOut is whether the computer with id and the labels has fails the rule.
func (r ComputerRule) LeftOut(id string, has func(label string) bool) bool {
	if len(r.IDs) > 0 && !slices.Contains(r.IDs, id) {
		return true
	}
	if len(r.LabelsIn) > 0 && !slices.ContainsFunc(r.LabelsIn, has) {
		return true
	}
	return slices.ContainsFunc(r.LabelsOut, has)
}
