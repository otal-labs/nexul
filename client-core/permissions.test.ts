import { describe, expect, it } from "vitest";

import { projectPermissions, workspaceWidePermissions, type MyWorkspaceInfo } from "@nexul/client-core/permissions";

const restricted: MyWorkspaceInfo = {
  role_name: "Client",
  permissions: ["chat:read"],
  restricted: true,
  projects: [
    { project_id: "p-web", actions: ["tickets:read", "tickets:write"] },
    { project_id: "p-api", actions: ["docs:read"] },
  ],
};

describe("a Restricted member's /me", () => {
  it("answers a project action from that project's access alone", () => {
    expect(projectPermissions(restricted, "p-web")).toEqual(["chat:read", "tickets:read", "tickets:write"]);
    expect(projectPermissions(restricted, "p-api")).not.toContain("tickets:read");
    expect(projectPermissions(restricted, "")).toEqual(workspaceWidePermissions(restricted));
  });

  it("opens an area held in any of their projects", () => {
    expect(workspaceWidePermissions(restricted)).toEqual(expect.arrayContaining(["tickets:read", "docs:read", "chat:read"]));
  });

  it("leaves an unrestricted member's permissions as the role gives them, on every project", () => {
    const member: MyWorkspaceInfo = { role_name: "Editor", permissions: ["tickets:read"] };
    expect(projectPermissions(member, "p-any")).toEqual(["tickets:read"]);
    expect(workspaceWidePermissions(member)).toEqual(["tickets:read"]);
  });
});

// Since projects became opt-in every non-Owner's /me reads as Restricted did, so a pre-switch client-core still answers per project.
describe("a non-Owner's /me after the switch, read by the client-core from before it", () => {
  const before: MyWorkspaceInfo = { role_name: "Editor", permissions: ["chat:write", "tickets:read", "tickets:write"] };
  const after: MyWorkspaceInfo = {
    role_name: "Editor, Frontend",
    roles: [
      { id: "role-editor", name: "Editor", is_owner: false },
      { id: "role-frontend", name: "Frontend", is_owner: false },
    ],
    permissions: ["chat:write"],
    restricted: true,
    projects: [
      { project_id: "p-web", actions: ["tickets:read", "tickets:write"] },
      { project_id: "p-api", actions: ["tickets:read", "tickets:write"] },
    ],
  };
  const sorted = (values: string[]) => [...new Set(values)].sort();

  it.each(["p-web", "p-api", undefined])("answers %s as before for a member listed in every project", (projectId) => {
    expect(sorted(projectPermissions(after, projectId))).toEqual(sorted(projectPermissions(before, projectId)));
    expect(sorted(workspaceWidePermissions(after))).toEqual(sorted(workspaceWidePermissions(before)));
  });

  it("answers a project's rule that takes an action away from that project alone", () => {
    const denied: MyWorkspaceInfo = { ...after, projects: [{ project_id: "p-web", actions: ["tickets:read"] }, ...after.projects!.slice(1)] };
    expect(projectPermissions(denied, "p-web")).not.toContain("tickets:write");
    expect(projectPermissions(denied, "p-api")).toContain("tickets:write");
  });
});
