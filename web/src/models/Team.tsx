import type { Provider } from "@/models/User";

// Mirrors internal/tenancy/model.go's Team wire shape (GET /api/team).
export const AccountStatus = {
  Active: "active",
  Disabled: "disabled",
  Removed: "removed",
} as const;

export type AccountStatus = (typeof AccountStatus)[keyof typeof AccountStatus];

export const getTeamKey = "getTeam";

// A removed field (ADR 0148): the server sends it empty and refuses a value.
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

export interface TeamMembership {
  workspace_id: string;
  workspace_name: string;
  role_id: string;
  role_name: string;
  // Every role held, in the order given; role_id and role_name name the first.
  roles?: TeamRole[];
  is_owner: boolean;
  allow: string[];
  deny: string[];
  every_project: EveryProject;
  // A removed field (ADR 0148), sent empty.
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
  // The linked sign-in accounts, in the order they were linked.
  providers: Provider[];
  // The name others know each of those accounts by; a provider without one (Google) is absent.
  usernames?: Partial<Record<Provider, string>>;
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

// Every role a membership holds, or the one role a server from before several roles names.
export const heldRoles = (membership: TeamMembership): TeamRole[] =>
  membership.roles ?? [{ id: membership.role_id, name: membership.role_name, is_owner: membership.is_owner }];

// The people holding a role in a workspace; removed accounts hold nothing.
export const roleHolders = (team: Team | undefined, workspaceId: string, roleId: string): TeamPerson[] =>
  (team?.people ?? []).filter(
    (person) =>
      person.status !== AccountStatus.Removed &&
      person.workspaces.some(
        (membership) =>
          membership.workspace_id === workspaceId &&
          heldRoles(membership).some((role) => role.id === roleId),
      ),
  );
