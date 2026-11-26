package pairing

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MaxLabels and maxLabelLength bound the labels an owner sets on one computer (ADR 0197).
const (
	MaxLabels      = 20
	maxLabelLength = 32
)

var labelPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// normalizeLabels lowercases and trims each label, drops blanks and repeats, and refuses one outside the rule.
func normalizeLabels(labels []string) ([]string, error) {
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label == "" || slices.Contains(out, label) {
			continue
		}
		if len(label) > maxLabelLength || !labelPattern.MatchString(label) {
			return nil, fmt.Errorf("%w: label %q: use lowercase letters, digits and dashes, up to %d characters", apperrs.ErrInvalid, label, maxLabelLength)
		}
		out = append(out, label)
	}
	if len(out) > MaxLabels {
		return nil, fmt.Errorf("%w: a computer takes up to %d labels", apperrs.ErrInvalid, MaxLabels)
	}
	return out, nil
}

// DerivedLabels are the labels a computer has from its facts alone: docker while Docker answers, and its OS.
func DerivedLabels(f *Facts) []string {
	out := []string{}
	if f == nil {
		return out
	}
	if f.Docker != nil && f.Docker.Available {
		out = append(out, "docker")
	}
	switch f.OS {
	case "linux", "windows":
		out = append(out, f.OS)
	case "darwin":
		out = append(out, "macos")
	}
	return out
}

// SetLabels replaces the labels on one of the caller's own computers; waiting runs look again, since a play's needs may now match.
func (s *Service) SetLabels(ctx context.Context, userID, computerID string, labels []string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	clean, err := normalizeLabels(labels)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetComputerLabels(ctx, userID, c.ID, clean, s.now().UTC()); err != nil {
		return nil, err
	}
	s.placementChanged(ctx, userID)
	c.Labels = clean
	c.BearerToken = ""
	return c, nil
}
