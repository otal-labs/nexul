import { z } from "zod";

import type { TeamRole } from "@/models/Team";

export const InvitationProvider = {
  GitHub: "github",
  Google: "google",
  Discord: "discord",
} as const;

export const getInvitationPreviewKey = "getInvitationPreview";

export type InvitationProvider = (typeof InvitationProvider)[keyof typeof InvitationProvider];

export interface InvitationGrant {
  workspace_id: string;
  workspace_name: string;
  // Every role the grant gives; role_id and role_name name the first.
  roles: Pick<TeamRole, "id" | "name">[];
  role_id: string;
  role_name: string;
}

export interface InvitationPreview {
  instance_name?: string;
  instance_url?: string;
  grants: InvitationGrant[];
  expires_at: string;
}

export interface InvitationAcceptance extends InvitationPreview {
  authenticated_user?: { id: string; login: string; name: string };
  acceptance_token?: string;
}

export interface ActiveInvitation {
  id: string;
  invited_by: string;
  created_at: string;
  // When the link stopped working, for an ended one.
  expires_at: string;
  grants: InvitationGrant[];
  redeemed_at?: string;
  redeemed_by?: string;
}

// The links the viewer may manage: those open, and those accepted or ended in the last 30 days, newest first.
export interface InvitationList {
  invitations: ActiveInvitation[];
  accepted: ActiveInvitation[];
  ended: ActiveInvitation[];
}

// "Acme · Editor, Frontend", how a link names what it gives.
export const invitationLinkLabel = (invitation: ActiveInvitation): string =>
  invitation.grants.map((grant) => `${grant.workspace_name} · ${grant.roles.map((role) => role.name).join(", ")}`).join("; ");

// "Junior Developer in Acme", how a person's row names what the link gave them.
export const invitationGrantsLabel = (invitation: ActiveInvitation): string =>
  invitation.grants.map((grant) => `${grant.roles.map((role) => role.name).join(", ")} in ${grant.workspace_name}`).join(", ");

export interface CreatedInvitation extends ActiveInvitation {
  url: string;
}

export interface RedeemedInvitation {
  token: string;
  workspace_ids: string[];
}

export const InvitationGrantFormSchema = z.object({
  workspace_id: z.string().min(1, "Choose a workspace"),
  role_ids: z.array(z.string()).min(1, "Pick at least one role"),
});

export type InvitationGrantInput = z.infer<typeof InvitationGrantFormSchema>;

export const CreateInvitationFormSchema = z.object({
  expires_in_days: z.union([z.literal(1), z.literal(7)]),
  grants: z.array(InvitationGrantFormSchema).min(1, "Add at least one workspace"),
});

export type CreateInvitationFormData = z.infer<typeof CreateInvitationFormSchema>;

export const hasDuplicateInvitationWorkspaces = (grants: readonly InvitationGrantInput[]): boolean =>
  new Set(grants.map((grant) => grant.workspace_id)).size !== grants.length;

export interface InvitationFragment {
  token: string;
  acceptance: boolean;
  malformed: boolean;
}

export const parseInvitationFragment = (hash: string): InvitationFragment => {
  const fragment = hash.startsWith("#") ? hash.slice(1) : hash;
  if (!fragment) return { token: "", acceptance: false, malformed: false };
  const acceptance = fragment.startsWith("acceptance-token=");
  const prefixedInvite = fragment.startsWith("invite-token=");
  const encodedToken = acceptance ? fragment.slice("acceptance-token=".length) : prefixedInvite ? fragment.slice("invite-token=".length) : fragment;
  try {
    return { token: decodeURIComponent(encodedToken), acceptance, malformed: false };
  } catch {
    return { token: "", acceptance: false, malformed: true };
  }
};
