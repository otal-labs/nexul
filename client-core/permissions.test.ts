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
