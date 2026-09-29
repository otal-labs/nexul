// Mirrors internal/tenancy/model.go's Team wire shape (GET /api/team).
export const AccountStatus = {
  Active: "active",
  Disabled: "disabled",
  Removed: "removed",
} as const;

export type AccountStatus = (typeof AccountStatus)[keyof typeof AccountStatus];

export const getTeamKey = "getTeam";

export interface TeamMembership {
  workspace_id: string;
  workspace_name: string;
  role_id: string;
  role_name: string;
  is_owner: boolean;
  allow: string[];
  deny: string[];
}

export interface TeamPerson {
  id: string;
  login: string;
  name: string;
  display_name?: string;
  avatar_url: string;
  status: AccountStatus;
  can_create_workspace: boolean;
  created_at: string;
  workspaces: TeamMembership[];
}

export interface TeamRole {
  id: string;
  name: string;
  is_owner: boolean;
}

export interface TeamWorkspace {
  id: string;
  name: string;
  can_manage_members: boolean;
  roles: TeamRole[];
}

export interface Team {
  people: TeamPerson[];
  workspaces: TeamWorkspace[];
}
