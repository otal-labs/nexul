// Package roles implements the Role entity: a permission catalog with a protected singleton Owner role.
package roles

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Role is one workspace's named permission set; the Owner role bypasses every check via IsOwnerRole, not its set.
type Role struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Permissions permissions.Set `json:"permissions"`
	IsOwnerRole bool            `json:"is_owner_role"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
