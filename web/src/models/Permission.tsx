import { z } from "zod";

// Mirrors the Go side's per-domain action set: read, write, delete, plus any verb a domain declares (ADR 0057).
export type PermissionAction = string;

// Where a domain applies (ADR 0097): inside one project, across the workspace, or instance-wide.
export const PermissionArea = {
  Project: "project",
  Workspace: "workspace",
  Instance: "instance",
} as const;

export type PermissionArea = (typeof PermissionArea)[keyof typeof PermissionArea];

// Served by GET /api/permissions/catalog: the full domain × action vocabulary the gateway enforces.
// Entries arrive grouped by domain, actions read/write/delete within a domain.
export interface PermissionInfo {
  value: string;
  label: string;
  domain: string;
  action: PermissionAction;
  area: PermissionArea;
}

// The one place client-side permission logic lives: a thin wrapper the server resolves, the client just reads.
export const hasPermission = (permissions: readonly string[] | undefined, value: string): boolean =>
  (permissions ?? []).includes(value);

// Mirrors internal/tenancy/handler.go's meResponse; a Restricted member's permissions carry no project area, so
// projects lists what they hold in each project they may open.
export interface MyWorkspaceInfo {
  role_name: string;
  permissions: string[];
  restricted?: boolean;
  projects?: { project_id: string; actions: string[] }[];
}

// What the viewer holds somewhere in the workspace, for the areas the sidebar and route gates open.
export const workspaceWidePermissions = (info: MyWorkspaceInfo | undefined): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  return [...info.permissions, ...(info.projects ?? []).flatMap((project) => project.actions)];
};

// What the viewer holds inside one project: the workspace areas plus, for a Restricted member, that project's access.
// A project they hold nothing on (none in view yet, or one just taken away) answers as any project would.
export const projectPermissions = (info: MyWorkspaceInfo | undefined, projectId: string): string[] => {
  if (!info?.restricted) return info?.permissions ?? [];
  const project = info.projects?.find((candidate) => candidate.project_id === projectId);
  if (!project) return workspaceWidePermissions(info);
  return [...info.permissions, ...project.actions];
};

export interface PermissionGrant {
  resource_type: string;
  resource_id: string;
  user_id: string;
  allow: string[];
  deny: string[];
  created_at: string;
  updated_at: string;
}

export interface PermissionUser {
  id: string;
  login: string;
  name: string;
}

// Two shapes on the wire: a doc grant (doc_ids, unchanged since before ticket 21) and a play exclusion
// (resource_type: "play", resource_ids) — kept apart so a doc submission never carries an extra field.
export const SetPermissionsSchema = z.union([
  z.object({
    doc_ids: z.array(z.string().min(1)).min(1, "Select at least one document"),
    user_ids: z.array(z.string().min(1)).min(1, "Select at least one user"),
    actions: z.array(z.string().min(1)).min(1, "Select at least one action"),
    grant: z.boolean(),
  }),
  z.object({
    resource_type: z.literal("play"),
    resource_ids: z.array(z.string().min(1)).min(1, "Select at least one play"),
    user_ids: z.array(z.string().min(1)).min(1, "Select at least one user"),
    actions: z.array(z.string().min(1)).min(1, "Select at least one action"),
    grant: z.boolean(),
  }),
]);

export type SetPermissionsInput = z.infer<typeof SetPermissionsSchema>;

// The sharing dialog's fields live outside the form (its own pickers), so the schema validates nothing.
export const PermissionsFormSchema = z.object({});

export type PermissionsFormData = z.infer<typeof PermissionsFormSchema>;
