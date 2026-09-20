import { z } from "zod";

export const InvitationProvider = {
  GitHub: "github",
  Google: "google",
  Discord: "discord",
} as const;

export const getInvitationPreviewKey = "getInvitationPreview";
export const getAccountsKey = "getAccounts";

export type InvitationProvider = (typeof InvitationProvider)[keyof typeof InvitationProvider];

export interface InvitationGrant {
  workspace_id: string;
  workspace_name: string;
  role_id: string;
  role_name: string;
  allow: string[];
  deny: string[];
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
  expires_at: string;
  grants: InvitationGrant[];
}

export interface CreatedInvitation extends ActiveInvitation {
  url: string;
}

export interface RedeemedInvitation {
  token: string;
  workspace_ids: string[];
}

export interface InvitationGrantInput {
  workspace_id: string;
  role_id: string;
  allow: string[];
  deny: string[];
}

export const InvitationGrantFormSchema = z
  .object({
    workspace_id: z.string().min(1, "Choose a workspace"),
    role_id: z.string().min(1, "Choose a role"),
    allow: z.array(z.string()),
    deny: z.array(z.string()),
  })
  .refine((grant) => grant.allow.every((value) => !grant.deny.includes(value)), {
    message: "A permission cannot be allowed and denied at the same time",
    path: ["allow"],
  });

export const CreateInvitationFormSchema = z.object({
  expires_in_days: z.union([z.literal(1), z.literal(7)]),
  grants: z.array(InvitationGrantFormSchema).min(1, "Add at least one workspace"),
});

export type CreateInvitationFormData = z.infer<typeof CreateInvitationFormSchema>;

export const hasDuplicateInvitationWorkspaces = (grants: readonly InvitationGrantInput[]): boolean =>
  new Set(grants.map((grant) => grant.workspace_id)).size !== grants.length;

export interface Account {
  id: string;
  provider: string;
  login: string;
  name: string;
  avatar_url: string;
  status: "active" | "disabled" | "removed";
  can_create_workspace: boolean;
  created_at: string;
}

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
