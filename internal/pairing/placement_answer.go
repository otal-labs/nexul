package pairing

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/labels"
)

// maxPlacementAnswers bounds the needs a project page asks about in one read, one per distinct set its plays name.
const maxPlacementAnswers = 20

// PlacementAnswer is where a run in a project needing Needs would land now, or the refusal and every computer it
// passed over; it is read from stored facts and asks no computer (spec decision 7).
type PlacementAnswer struct {
	Needs      []string   `json:"needs"`
	ComputerID string     `json:"computer_id,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	PassedOver []PassOver `json:"passed_over"`
}

// PlacementAnswers answers, for userID's runs in projectID, where each set of needs would land now; no set asks once for
// a play that needs nothing.
func (s *Service) PlacementAnswers(ctx context.Context, userID, projectID string, needsSets [][]string) ([]PlacementAnswer, error) {
	if len(needsSets) == 0 {
		needsSets = [][]string{nil}
	}
	if len(needsSets) > maxPlacementAnswers {
		return nil, fmt.Errorf("%w: ask about at most %d sets of needs at once", apperrs.ErrInvalid, maxPlacementAnswers)
	}
	out := make([]PlacementAnswer, 0, len(needsSets))
	for _, raw := range needsSets {
		needs, err := labels.Normalize(raw)
		if err != nil {
			return nil, err
		}
		answer, err := s.placementAnswer(ctx, userID, projectID, needs)
		if err != nil {
			return nil, err
		}
		out = append(out, answer)
	}
	return out, nil
}

func (s *Service) placementAnswer(ctx context.Context, userID, projectID string, needs []string) (PlacementAnswer, error) {
	answer := PlacementAnswer{Needs: needs, PassedOver: []PassOver{}}
	target, err := s.PreviewRun(ctx, RunRequest{UserID: userID, ProjectID: projectID, Needs: needs, Room: RoomPreferred, Person: true})
	var nc *NotConfiguredError
	if errors.As(err, &nc) {
		answer.Reason = string(nc.Reason)
		answer.PassedOver = append(answer.PassedOver, nc.PassedOver...)
		return answer, nil
	}
	if err != nil {
		return PlacementAnswer{}, err
	}
	answer.ComputerID = target.Computer.ID
	answer.PassedOver = append(answer.PassedOver, target.PassedOver...)
	return answer, nil
}
