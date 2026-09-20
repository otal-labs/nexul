import { describe, expect, it } from "vitest";

import { CreateInvitationFormSchema, hasDuplicateInvitationWorkspaces } from "@/models/Invitation";

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
});
