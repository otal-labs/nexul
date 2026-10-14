// Package roles implements the Role entity: a permission catalog with a protected singleton Owner role.
package roles

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Role is one workspace's named permission set; the Owner role bypasses every check via IsOwnerRole, not its set.
type Role struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Permissions permissions.Set `json:"permissions"`
	IsOwnerRole bool            `json:"is_owner_role"`
	// WelcomeSteps are the Welcome wizard steps the role gives its holders, as welcomestep keys.
	WelcomeSteps []string `json:"welcome_steps"`
	// ShowLockedActions shows its holders an action they can't take dimmed with the reason, instead of leaving it out (ADR 0152).
	ShowLockedActions bool      `json:"show_locked_actions"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DescriptionMax is the most characters a role's one-line description holds.
const DescriptionMax = 140

// checkDescription trims a role's description and refuses one longer than DescriptionMax characters.
func checkDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if n := utf8.RuneCountInString(description); n > DescriptionMax {
		return "", fmt.Errorf("%w: a role's description holds at most %d characters, this one has %d", apperrs.ErrInvalid, DescriptionMax, n)
	}
	return description, nil
}
