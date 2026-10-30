package automations

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// VersionInfo is a version in the history dialog's shared shape: the sequence is the version number, the push message its
// name and its pusher the one editor. ID and Status are the automation's own, so the dialog can merge or roll back.
type VersionInfo struct {
	ID        string        `json:"id"`
	Version   int           `json:"version"`
	CreatedAt time.Time     `json:"created_at"`
	Name      string        `json:"name"`
	EditorIDs []string      `json:"editor_ids"`
	Via       string        `json:"via"`
	Current   bool          `json:"current"`
	Status    VersionStatus `json:"status"`
	// Automations keep no word counts or restores: a rollback repoints the active version and writes none.
	WordsAdded   *int `json:"words_added"`
	WordsRemoved *int `json:"words_removed"`
	RestoredFrom *int `json:"restored_from"`
}

// VersionBody is a version with its code as the body.
type VersionBody struct {
	VersionInfo
	Body string `json:"body"`
}

// History is an automation's versions newest first, without code; automations keep no field changes.
type History struct {
	Versions []VersionInfo `json:"versions"`
	Changes  []struct{}    `json:"changes"`
}

// seedVia is how the history credits a version the seeding hook landed.
const seedVia = "Nexul"

func versionInfo(v Version) VersionInfo {
	info := VersionInfo{
		ID:        v.ID,
		Version:   v.Sequence,
		CreatedAt: v.CreatedAt,
		Name:      v.Message,
		EditorIDs: []string{v.PusherID},
		Current:   v.Status == VersionActive,
		Status:    v.Status,
	}
	if v.PusherID == seedPusherID {
		info.EditorIDs = []string{}
		info.Via = seedVia
	}
	return info
}

// History returns every version of automationID newest first, without code. None is pruned, so the list holds every
// version a push or a seed ever landed except a pending one a later push replaced.
func (s *VersionsService) History(ctx context.Context, actorID, automationID string) (*History, error) {
	if _, err := requireOn(ctx, s.perm, s.automations, actorID, automationID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	list, err := s.repo.ListSummaries(ctx, automationID)
	if err != nil {
		return nil, fmt.Errorf("history of automation %s: %w", automationID, err)
	}
	out := History{Versions: make([]VersionInfo, 0, len(list)), Changes: []struct{}{}}
	for _, v := range list {
		out.Versions = append(out.Versions, versionInfo(v))
	}
	return &out, nil
}

// VersionBody returns one version by its sequence with its code as the body.
func (s *VersionsService) VersionBody(ctx context.Context, actorID, automationID string, sequence int) (*VersionBody, error) {
	if _, err := requireOn(ctx, s.perm, s.automations, actorID, automationID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	if sequence < 1 {
		return nil, fmt.Errorf("%w: version must be a number from 1 up", apperrs.ErrInvalid)
	}
	v, err := s.repo.GetBySequence(ctx, automationID, sequence)
	if err != nil {
		return nil, fmt.Errorf("get version %d of automation %s: %w", sequence, automationID, err)
	}
	return &VersionBody{VersionInfo: versionInfo(*v), Body: v.Code}, nil
}
