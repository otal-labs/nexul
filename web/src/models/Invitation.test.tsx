import { describe, expect, it } from "vitest";

import { CreateInvitationFormSchema, hasDuplicateInvitationWorkspaces, invitationRequest, parseInvitationFragment } from "@/models/Invitation";

describe("CreateInvitationFormSchema", () => {
  it("accepts one or more grants with the fixed expiry choices", () => {
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 7, grants: [{ workspace_id: "ws-1", role_id: "role-1", allow: [], deny: [] }] }).success).toBe(true);
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 30, grants: [{ workspace_id: "ws-1", role_id: "role-1", allow: [], deny: [] }] }).success).toBe(false);
  });

  it("rejects an override that appears in both allow and deny", () => {
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 1, grants: [{ workspace_id: "ws-1", role_id: "role-1", allow: ["docs:read"], deny: ["docs:read"] }] }).success).toBe(false);
  });

  it("detects duplicate workspace grants before submission", () => {
    expect(hasDuplicateInvitationWorkspaces([{ workspace_id: "ws-1", role_id: "r-1", allow: [], deny: [] }, { workspace_id: "ws-1", role_id: "r-2", allow: [], deny: [] }])).toBe(true);
  });

  it("reports malformed fragments without throwing", () => {
    expect(parseInvitationFragment("#%E0%A4%A")).toEqual({ token: "", acceptance: false, malformed: true });
    expect(parseInvitationFragment("#acceptance-token=abc")).toEqual({ token: "abc", acceptance: true, malformed: false });
  });
});

describe("invitationRequest", () => {
  const grant = { workspace_id: "ws-1", role_id: "r-1", allow: [], deny: [] };
  const access = [{ project_id: "p-1", allow: ["tickets:read"] }, { project_id: "p-2", allow: [] }];

  it("sends the chosen projects only under Chosen projects, without the ones taken back to None", () => {
    const sent = invitationRequest({ expires_in_days: 7, grants: [{ ...grant, every_project: "none", project_access: access }] });
    expect(sent.grants[0]?.project_access).toEqual([{ project_id: "p-1", allow: ["tickets:read"] }]);
  });

  it("drops levels picked before switching back to From role", () => {
    const sent = invitationRequest({ expires_in_days: 7, grants: [{ ...grant, every_project: "role", project_access: access }] });
    expect(sent.grants[0]?.project_access).toEqual([]);
  });
});
