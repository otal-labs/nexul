// @vitest-environment node
import { describe, expect, it } from "vitest";

import { projectPermissions, workspaceWidePermissions, type MyWorkspaceInfo } from "@nexul/client-core/permissions";

const member: MyWorkspaceInfo = {
  role_name: "Client",
  roles: [{ id: "role-client", name: "Client", is_owner: false }],
  permissions: ["chat:read", "projects:create"],
  projects: [
    { project_id: "p-web", actions: ["tickets:read", "tickets:write"] },
    { project_id: "p-api", actions: ["docs:read"] },
  ],
};

describe("a member's /me", () => {
  it("answers a project action from what they hold in that project alone", () => {
    expect(projectPermissions(member, "p-web")).toEqual(["chat:read", "projects:create", "tickets:read", "tickets:write"]);
    expect(projectPermissions(member, "p-api")).not.toContain("tickets:read");
    expect(projectPermissions(member, "")).toEqual(workspaceWidePermissions(member));
  });

  it("opens an area held in any of their projects", () => {
    expect(workspaceWidePermissions(member)).toEqual(expect.arrayContaining(["tickets:read", "docs:read", "chat:read"]));
  });

  it("holds no project area while no project lists them, whatever their roles grant", () => {
    const unlisted: MyWorkspaceInfo = { ...member, projects: [] };
    expect(workspaceWidePermissions(unlisted)).toEqual(["chat:read", "projects:create"]);
    expect(projectPermissions(unlisted, "p-web")).not.toContain("tickets:read");
  });

  it("answers the Owner's permissions on every project", () => {
    const owner: MyWorkspaceInfo = {
      role_name: "Owner",
      roles: [{ id: "role-owner", name: "Owner", is_owner: true }],
      permissions: ["tickets:read", "tickets:write"],
      projects: [],
    };
    expect(projectPermissions(owner, "p-any")).toEqual(["tickets:read", "tickets:write"]);
  });
});

// HTTP only grows (ADR 0082): a /me shaped as before the switch and one shaped after it give the same answers.
describe("a /me from before the switch and one from after it", () => {
  const before: MyWorkspaceInfo = { role_name: "Editor", permissions: ["chat:write", "tickets:read", "tickets:write"] };
  const after: MyWorkspaceInfo = {
    role_name: "Editor, Frontend",
    roles: [
      { id: "role-editor", name: "Editor", is_owner: false },
      { id: "role-frontend", name: "Frontend", is_owner: false },
    ],
    permissions: ["chat:write"],
    projects: [
      { project_id: "p-web", actions: ["tickets:read", "tickets:write"] },
      { project_id: "p-api", actions: ["tickets:read", "tickets:write"] },
    ],
  };
  const sorted = (values: string[]) => [...new Set(values)].sort();

  it.each(["p-web", "p-api", undefined])("answers %s alike for a member listed in every project", (projectId) => {
    expect(sorted(projectPermissions(after, projectId))).toEqual(sorted(projectPermissions(before, projectId)));
    expect(sorted(workspaceWidePermissions(after))).toEqual(sorted(workspaceWidePermissions(before)));
  });

  it("answers a project's rule that takes an action away from that project alone", () => {
    const denied: MyWorkspaceInfo = { ...after, projects: [{ project_id: "p-web", actions: ["tickets:read"] }, ...after.projects!.slice(1)] };
    expect(projectPermissions(denied, "p-web")).not.toContain("tickets:write");
    expect(projectPermissions(denied, "p-api")).toContain("tickets:write");
  });
});
