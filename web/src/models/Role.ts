export interface Role {
  id: string;
  workspace_id: string;
  name: string;
  permissions: string[];
  is_owner_role: boolean;
  created_at: string;
  updated_at: string;
}
