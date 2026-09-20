import { z } from "zod";

// Mirrors the Go side's per-domain action set: read, write, delete, plus any verb a domain declares (ADR 0057).
export type PermissionAction = string;

// Served by GET /api/permissions/catalog: the full domain × action vocabulary the gateway enforces.
// Entries arrive grouped by domain, actions read/write/delete within a domain.
export interface PermissionInfo {
  value: string;
  label: string;
  domain: string;
  action: PermissionAction;
}

// The one place client-side permission logic lives: a thin wrapper the server resolves, the client just reads.
export const hasPermission = (permissions: readonly string[] | undefined, value: string): boolean =>
  (permissions ?? []).includes(value);

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
  can_create_workspace: boolean;
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
