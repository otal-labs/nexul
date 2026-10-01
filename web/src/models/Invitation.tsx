import { z } from "zod";

import { EveryProject, type ProjectGrant } from "@/models/Team";

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
  role_id: string;
  role_name: string;
  allow: string[];
  deny: string[];
  every_project?: EveryProject;
  project_access?: ProjectGrant[];
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
  every_project?: EveryProject | undefined;
  project_access?: ProjectGrant[] | undefined;
}

export const InvitationGrantFormSchema = z
  .object({
    workspace_id: z.string().min(1, "Choose a workspace"),
    role_id: z.string().min(1, "Choose a role"),
    allow: z.array(z.string()),
    deny: z.array(z.string()),
    // Omitted means From role; under None the person lands restricted to project_access.
    every_project: z.enum([EveryProject.Role, EveryProject.None]).optional(),
    project_access: z.array(z.object({ project_id: z.string().min(1), allow: z.array(z.string()) })).optional(),
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

// Project access only travels with None; levels picked before switching back to From role stay behind.
export const invitationRequest = (input: CreateInvitationFormData): CreateInvitationFormData => ({
  ...input,
  grants: input.grants.map((grant) => ({
    ...grant,
    project_access: grant.every_project === EveryProject.None ? (grant.project_access ?? []).filter((access) => access.allow.length > 0) : [],
  })),
});

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
