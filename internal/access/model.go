package access

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Overwrite is workspace-wide when ResourceType is "workspace" (ResourceID is the workspace id).
type Overwrite struct {
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	UserID       string          `json:"user_id"`
	Allow        permissions.Set `json:"allow"`
	Deny         permissions.Set `json:"deny"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// User is mirrored here so access never imports auth (ADR 0017).
type User struct {
	ID                 string `json:"id"`
	Login              string `json:"login"`
	Name               string `json:"name"`
	CanCreateWorkspace bool   `json:"can_create_workspace"`
}

// RoleInfo is mirrored here so access never imports roles (ADR 0017).
type RoleInfo struct {
	IsOwnerRole bool
	Permissions permissions.Set
}
