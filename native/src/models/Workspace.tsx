// Mirrors internal/tenancy/model.go's Workspace wire shape, trimmed to the fields this app reads.
export interface Workspace {
  id: string;
  name: string;
}

// The server's /me answer: an Owner's list is the whole grid, so the Owner never needs a check of its own.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
}
