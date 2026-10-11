import { describe, expect, it } from "vitest";

import { AccountStatus, EveryProject, roleHolders, type Team, type TeamMembership, type TeamPerson } from "@/models/Team";

const membership = (roleId: string, roles?: string[]): TeamMembership => ({
  workspace_id: "ws-1",
  workspace_name: "Acme",
  role_id: roleId,
  role_name: roleId,
  ...(roles && { roles: roles.map((id) => ({ id, name: id, is_owner: false })) }),
  is_owner: false,
  allow: [],
  deny: [],
  every_project: EveryProject.Role,
  projects: [],
});

const person = (id: string, workspaces: TeamMembership[], status: AccountStatus = AccountStatus.Active): TeamPerson => ({
  id,
  login: id,
  name: id,
  avatar_url: "",
  status,
  created_at: "2026-10-01T00:00:00Z",
  providers: [],
  online: false,
  last_seen_at: null,
  workspaces,
});

describe("roleHolders", () => {
  it("counts whoever holds the role among their roles, and not a removed account", () => {
    const team: Team = {
      people: [
        person("alice", [membership("role-dev", ["role-dev", "role-client"])]),
        person("bob", [membership("role-client")]),
        person("sam", [membership("role-dev")]),
        person("lena", [membership("role-client")], AccountStatus.Removed),
      ],
      workspaces: [],
      can_manage_accounts: false,
    };

    expect(roleHolders(team, "ws-1", "role-client").map((p) => p.id)).toEqual(["alice", "bob"]);
  });
});
