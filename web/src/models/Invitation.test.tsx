import { describe, expect, it } from "vitest";

import { CreateInvitationFormSchema, hasDuplicateInvitationWorkspaces, parseInvitationFragment } from "@/models/Invitation";

describe("CreateInvitationFormSchema", () => {
  it("accepts one or more grants with the fixed expiry choices", () => {
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 7, grants: [{ workspace_id: "ws-1", role_ids: ["role-1", "role-2"] }] }).success).toBe(true);
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 30, grants: [{ workspace_id: "ws-1", role_ids: ["role-1"] }] }).success).toBe(false);
  });

  it("refuses a workspace with no role, since an invitation gives a role on joining", () => {
    expect(CreateInvitationFormSchema.safeParse({ expires_in_days: 1, grants: [{ workspace_id: "ws-1", role_ids: [] }] }).success).toBe(false);
  });

  it("detects duplicate workspace grants before submission", () => {
    expect(hasDuplicateInvitationWorkspaces([{ workspace_id: "ws-1", role_ids: ["r-1"] }, { workspace_id: "ws-1", role_ids: ["r-2"] }])).toBe(true);
  });

  it("reports malformed fragments without throwing", () => {
    expect(parseInvitationFragment("#%E0%A4%A")).toEqual({ token: "", acceptance: false, malformed: true });
    expect(parseInvitationFragment("#acceptance-token=abc")).toEqual({ token: "abc", acceptance: true, malformed: false });
  });
});
