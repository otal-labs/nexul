package pairing

import (
	"context"
	"slices"

	"github.com/otal-labs/nexul/internal/platform/labels"
)

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
func (s *Service) SetLabels(ctx context.Context, userID, computerID string, in []string) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	clean, err := labels.Normalize(in)
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

// hasLabel is whether label is among c's own labels or its facts'.
func hasLabel(c Computer, label string) bool {
	return slices.Contains(c.Labels, label) || slices.Contains(DerivedLabels(c.Facts), label)
}

// missingNeed is the first of needs c lacks among its own and its facts' labels; empty when it has them all.
func missingNeed(c Computer, needs []string) string {
	for _, need := range needs {
		if !hasLabel(c, need) {
			return need
		}
	}
	return ""
}
