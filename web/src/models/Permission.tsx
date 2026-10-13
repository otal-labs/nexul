import { z } from "zod";

// Mirrors the Go side's per-domain action set: read, write, delete, plus any verb a domain declares (ADR 0057).
export type PermissionAction = string;

// Where a domain applies (ADR 0148): inside one project, across the workspace, or instance-wide.
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

export interface PermissionGrant {
  resource_type: string;
  resource_id: string;
  // Whom the rule names: a person by user_id, or a role by role_id (ADR 0148).
  target: "person" | "role";
  user_id: string;
  role_id?: string;
  allow: string[];
  deny: string[];
  created_at: string;
  updated_at: string;
}

// What a rule holds for one permission (ADR 0148): Fallback says nothing, so the answer comes from further out.
export const RuleState = {
  Allow: "allow",
  Fallback: "fallback",
  Deny: "deny",
} as const;

export type RuleState = (typeof RuleState)[keyof typeof RuleState];

// What decided an answer, with its names resolved and the sentence a person reads (GET /api/permissions/explain).
export interface PermissionSource {
  kind: string;
  entity_type: string;
  entity_id: string;
  target_type: string;
  target_id: string;
  state: string;
  entity_name: string;
  target_name: string;
  sentence: string;
}

export interface PermissionVerdict {
  allowed: boolean;
  source: PermissionSource;
}

// One permission's answer, and the answer the entity would give with its own rule for the person at Fallback.
export interface PermissionAnswer extends PermissionVerdict {
  action: string;
  fallback: PermissionVerdict;
}

export interface PermissionExplanation {
  resource_type: string;
  resource_id: string;
  user_id: string;
  // Set instead of user_id when the answers are a role's, as a holder of it alone would get them.
  role_id?: string;
  effective: PermissionAnswer[];
}

// One PUT /api/permissions: one target's actions moved to one state on one doc, play or project.
export interface RuleChange {
  // A targetKey: "role:<id>" or "person:<id>".
  target: string;
  state: RuleState;
  actions: string[];
}

// The panel's held rules, keyed by ruleKey(target, action); a key absent keeps what is saved.
export const PermissionRulesFormSchema = z.object({
  rules: z.record(z.string(), z.enum([RuleState.Allow, RuleState.Fallback, RuleState.Deny])),
});

export type PermissionRulesFormData = z.infer<typeof PermissionRulesFormSchema>;
