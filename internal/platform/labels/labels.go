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
