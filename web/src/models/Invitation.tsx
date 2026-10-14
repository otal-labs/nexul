import type { AxiosError } from "axios";
import { z } from "zod";

import type { ApiErrorBody } from "@/api/client";
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
  authenticated_user?: RefusedPerson & { id: string };
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

// The account a link turned away, as the provider or the session named it.
export interface RefusedPerson {
  provider: InvitationProvider;
  login: string;
  name: string;
  avatar_url?: string;
}

// Why the invite page shows a refusal instead of the invitation: the link is gone, or the person already has an account.
export type InvitationRefusal = { kind: "gone" } | { kind: "existing_account"; person?: RefusedPerson | undefined };

export const goneRefusal: InvitationRefusal = { kind: "gone" };

// A 404 is every way a link stops working (expired, deleted, used); existing_account is the server turning an account away.
export const invitationRefusal = (error: unknown): InvitationRefusal | undefined => {
  const response = (error as AxiosError<ApiErrorBody> | null)?.response;
  if (response?.status === 404) return goneRefusal;
  if (response?.data?.code !== "existing_account") return undefined;
  return { kind: "existing_account", person: response.data.details as RefusedPerson | undefined };
};

export interface InvitationFragment {
  token: string;
  acceptance: boolean;
  malformed: boolean;
  // Set when the provider round trip came back refused; the fragment then carries no token.
  refusal?: InvitationRefusal;
}

const refusalFromFragment = (fragment: string): InvitationRefusal => {
  const params = new URLSearchParams(fragment);
  if (params.get("refused") !== "existing_account") return goneRefusal;
  const provider = params.get("provider");
  const known = Object.values(InvitationProvider).find((value) => value === provider);
  const login = params.get("login") ?? "";
  return { kind: "existing_account", person: known && login ? { provider: known, login, name: params.get("name") ?? "" } : undefined };
};

export const parseInvitationFragment = (hash: string): InvitationFragment => {
  const fragment = hash.startsWith("#") ? hash.slice(1) : hash;
  if (!fragment) return { token: "", acceptance: false, malformed: false };
  if (new URLSearchParams(fragment).has("refused")) return { token: "", acceptance: false, malformed: false, refusal: refusalFromFragment(fragment) };
  const acceptance = fragment.startsWith("acceptance-token=");
  const prefixedInvite = fragment.startsWith("invite-token=");
  const encodedToken = acceptance ? fragment.slice("acceptance-token=".length) : prefixedInvite ? fragment.slice("invite-token=".length) : fragment;
  try {
    return { token: decodeURIComponent(encodedToken), acceptance, malformed: false };
  } catch {
    return { token: "", acceptance: false, malformed: true };
  }
};
