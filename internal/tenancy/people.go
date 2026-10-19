package tenancy

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ListPeople needs only membership of workspaceID, never members:write: seeing who you work with is not managing them (ADR 0086).
func (s *Service) ListPeople(ctx context.Context, actorID, workspaceID string) ([]Person, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.requireMember(ctx, actorID, workspaceID); err != nil {
		return nil, err
	}
	members, err := s.members.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for workspace %s: %w", workspaceID, err)
	}
	userIDs := make([]string, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}
	people, err := s.peopleNamed(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	owners, err := s.ownerIDs(ctx, workspaceID, members)
	if err != nil {
		return nil, err
	}
	for i := range people {
		people[i].Owner = owners[people[i].UserID]
	}
	return people, nil
}

// ownerIDs is the members holding workspaceID's Owner role, asking once per role.
func (s *Service) ownerIDs(ctx context.Context, workspaceID string, members []*Member) (map[string]bool, error) {
	isOwner := map[string]bool{}
	owners := map[string]bool{}
	for _, m := range members {
		for _, roleID := range m.RoleIDs {
			owner, seen := isOwner[roleID]
			if !seen {
				var err error
				if owner, err = s.roleNames.IsOwnerRole(ctx, workspaceID, roleID); err != nil {
					return nil, fmt.Errorf("check owner role %s: %w", roleID, err)
				}
				isOwner[roleID] = owner
			}
			if owner {
				owners[m.UserID] = true
			}
		}
	}
	return owners, nil
}

// SetProjects wires the project lookups a project's people are read through.
func (s *Service) SetProjects(g ProjectGate) { s.projects = g }

// ListProjectPeople is the workspace's People who are in projectID, for a ticket's developer and tester pickers;
// anyone in the project reads it.
func (s *Service) ListProjectPeople(ctx context.Context, actorID, projectID string) ([]Person, error) {
	workspaceID, err := s.openProject(ctx, actorID, projectID)
	if err != nil {
		return nil, err
	}
	userIDs, err := s.members.ProjectMemberIDs(ctx, workspaceID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list the people of project %s: %w", projectID, err)
	}
	return s.peopleNamed(ctx, userIDs)
}

// ListProjectRoles is the roles projectID's rules let in, for a doc's permissions panel, which lists only those;
// anyone in the project reads it.
func (s *Service) ListProjectRoles(ctx context.Context, actorID, projectID string) ([]string, error) {
	if _, err := s.openProject(ctx, actorID, projectID); err != nil {
		return nil, err
	}
	return s.members.ProjectRoleIDs(ctx, projectID)
}

// openProject is projectID's workspace, when actorID is in the project; otherwise it is not found.
func (s *Service) openProject(ctx context.Context, actorID, projectID string) (string, error) {
	if s.projects == nil {
		return "", fmt.Errorf("%w: no project lookup wired", apperrs.ErrForbidden)
	}
	workspaceID, err := s.projects.ProjectWorkspace(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("resolve project %s: %w", projectID, err)
	}
	if _, opens := s.projects.ProjectPermissions(ctx, actorID, projectID); !opens {
		return "", fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	return workspaceID, nil
}

// peopleNamed is the Person of each of userIDs with an account, in their order.
func (s *Service) peopleNamed(ctx context.Context, userIDs []string) ([]Person, error) {
	accounts, err := s.accounts.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	byID := make(map[string]*TeamAccount, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}
	people := make([]Person, 0, len(userIDs))
	for _, id := range userIDs {
		a, ok := byID[id]
		if !ok {
			continue
		}
		people = append(people, toPerson(a))
	}
	return people, nil
}

func (s *Service) requireMember(ctx context.Context, actorID, workspaceID string) error {
	_, err := s.members.RoleIDFor(ctx, workspaceID, actorID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w: only members of this workspace see its people", apperrs.ErrForbidden)
	}
	if err != nil {
		return fmt.Errorf("check membership of workspace %s: %w", workspaceID, err)
	}
	return nil
}

func toPerson(a *TeamAccount) Person {
	name := a.DisplayName
	if name == "" {
		name = a.Name
	}
	return Person{UserID: a.ID, Login: a.Login, DisplayName: name, AvatarURL: effectiveAvatarURL(a), Providers: a.Providers, Usernames: a.Usernames}
}

// effectiveAvatarURL versions an uploaded picture by its content, so no cache can serve the old one after a new upload.
func effectiveAvatarURL(a *TeamAccount) string {
	if a.AvatarOverride == "" {
		return a.AvatarURL
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(a.AvatarOverride)) // a hash.Hash never returns a write error
	return fmt.Sprintf("/api/people/%s/avatar?v=%x", a.ID, h.Sum64())
}

// Avatar returns a person's uploaded picture to themselves, to anyone sharing a workspace with them, and to a holder of accounts:read.
func (s *Service) Avatar(ctx context.Context, actorID, userID string) (contentType string, data []byte, err error) {
	userID = strings.TrimSpace(userID)
	if err := s.requireSeesPerson(ctx, actorID, userID); err != nil {
		return "", nil, err
	}
	a, err := s.accounts.Account(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("get account %s: %w", userID, err)
	}
	if a.AvatarOverride == "" {
		return "", nil, fmt.Errorf("%w: %s has not uploaded a picture", apperrs.ErrNotFound, a.Login)
	}
	return decodeAvatar(a.AvatarOverride)
}

func (s *Service) requireSeesPerson(ctx context.Context, actorID, userID string) error {
	if actorID != "" && actorID == userID {
		return nil
	}
	readsAccounts, err := s.perm.HoldsAnywhere(ctx, actorID, permissions.AccountsRead)
	if err != nil {
		return fmt.Errorf("check accounts:read: %w", err)
	}
	if readsAccounts {
		return nil
	}
	workspaces, err := s.repo.ListForUser(ctx, actorID)
	if err != nil {
		return fmt.Errorf("list workspaces for %s: %w", actorID, err)
	}
	for _, w := range workspaces {
		_, err := s.members.RoleIDFor(ctx, w.ID, userID)
		if err == nil {
			return nil
		}
		if !errors.Is(err, apperrs.ErrNotFound) {
			return fmt.Errorf("check membership of workspace %s: %w", w.ID, err)
		}
	}
	return fmt.Errorf("%w: you share no workspace with this person", apperrs.ErrForbidden)
}

// decodeAvatar serves only raster images as themselves; anything else goes out as opaque bytes.
func decodeAvatar(dataURI string) (string, []byte, error) {
	meta, payload, ok := strings.Cut(strings.TrimPrefix(dataURI, "data:"), ",")
	contentType, isBase64 := strings.CutSuffix(meta, ";base64")
	if !ok || !isBase64 {
		return "", nil, fmt.Errorf("stored avatar is not a base64 data URI")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, fmt.Errorf("decode stored avatar: %w", err)
	}
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
		return contentType, data, nil
	}
	return "application/octet-stream", data, nil
}
