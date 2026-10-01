// Mirrors internal/tenancy/model.go's Team wire shape (GET /api/team).
export const AccountStatus = {
  Active: "active",
  Disabled: "disabled",
  Removed: "removed",
} as const;

export type AccountStatus = (typeof AccountStatus)[keyof typeof AccountStatus];

export const getTeamKey = "getTeam";

// A membership's Every project row: From role, or None for a Restricted member (ADR 0097).
export const EveryProject = {
  Role: "role",
  None: "none",
} as const;

export type EveryProject = (typeof EveryProject)[keyof typeof EveryProject];

// One project's levels, held as the actions they expand to; an empty allow takes the project away.
export interface ProjectGrant {
  project_id: string;
  project_name?: string;
  allow: string[];
}

// The levels per project keyed by project, the shape the access rows read.
export const accessByProject = (grants: readonly ProjectGrant[]): Record<string, string[]> =>
  Object.fromEntries(grants.map((grant) => [grant.project_id, grant.allow]));

// Replaces one project's levels in a list of grants, dropping the project once nothing is left.
export const withProjectGrant = (grants: readonly ProjectGrant[], projectId: string, allow: string[]): ProjectGrant[] => {
  const rest = grants.filter((grant) => grant.project_id !== projectId);
  if (allow.length === 0) return rest;
  return [...rest, { project_id: projectId, allow }];
};

export interface TeamMembership {
  workspace_id: string;
  workspace_name: string;
  role_id: string;
  role_name: string;
  is_owner: boolean;
  allow: string[];
  deny: string[];
  every_project: EveryProject;
  // Only projects the viewer may open themselves, kept while Every project is From role.
  projects: ProjectGrant[];
}

export interface TeamPerson {
  id: string;
  login: string;
  name: string;
  display_name?: string;
  avatar_url: string;
  status: AccountStatus;
  created_at: string;
  // A live browser socket right now; last_seen_at is the latest session activity, to the hour, null once signed out everywhere.
  online: boolean;
  last_seen_at: string | null;
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
  // Whether the viewer holds accounts:write in any workspace; the web reads the finer bits from /me instead.
  can_manage_accounts: boolean;
}
