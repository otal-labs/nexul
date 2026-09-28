// Mirrors internal/tenancy/model.go's Workspace wire shape, trimmed to the fields this app reads.
export interface Workspace {
  id: string;
  name: string;
}
