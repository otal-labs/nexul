import { z } from "zod";

// Mirrors internal/tenancy/model.go's MemberView/Invite/MembersList wire shapes (Membership invites' Members page).
export interface MemberView {
  user_id: string;
  login: string;
  role_id: string;
}

export interface WorkspaceInvite {
  workspace_id: string;
  login: string;
  role_id: string;
  invited_by: string;
  created_at: string;
}

export interface MembersList {
  members: MemberView[];
  invites: WorkspaceInvite[];
}

export const InviteMemberFormSchema = z.object({
  login: z.string().trim().min(1, "GitHub username is required"),
});

export type InviteMemberFormData = z.infer<typeof InviteMemberFormSchema>;

// One row in the bulk-invite pending list: a login queued for submission with its picked role.
export interface PendingInvite {
  login: string;
  roleId: string;
}
