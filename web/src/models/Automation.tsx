import { z } from "zod";

import { AutomationRunOutcome } from "@/enums/Automation";
import type { AutomationKind } from "@/enums/Automation";

// Name/description/subscriptions/config_schema are code-declared and overwritten by the SDK on every dial-in.
export interface Automation {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  kind: AutomationKind;
  enabled: boolean;
  subscriptions: string[];
  config_schema: unknown;
  config_values: unknown;
  scopes: string[];
  token_prefix?: string;
  token_revoked_at?: string;
  // The automations host it runs on; null means the bundled instance host.
  host_id: string | null;
  created_at: string;
  updated_at: string;
  // Read with the list and the record; absent on what a write returns.
  last_run?: AutomationLastRun | null;
  runs_7d?: AutomationRunCounts;
  pending_version?: number | null;
}

export interface AutomationLastRun {
  outcome: AutomationRunOutcome;
  started_at: string;
  error?: string;
}

// Failed counts failures and crashes.
export interface AutomationRunCounts {
  total: number;
  failed: number;
}

export const AutomationState = {
  Failing: "failing",
  Attention: "attention",
  On: "on",
  Off: "off",
} as const;

export type AutomationState = (typeof AutomationState)[keyof typeof AutomationState];

export const AutomationAttention = {
  HostOffline: "host_offline",
  NeedsConfiguration: "needs_configuration",
  TokenRevoked: "token_revoked",
  PendingVersion: "pending_version",
} as const;

export type AutomationAttention = (typeof AutomationAttention)[keyof typeof AutomationAttention];

// The most urgent thing waiting on someone, or null; the instance host (no host_id) is never offline.
export const automationAttention = (automation: Automation, offlineHostIds: ReadonlySet<string>): AutomationAttention | null => {
  if (automation.host_id && offlineHostIds.has(automation.host_id)) return AutomationAttention.HostOffline;
  if (automationNeedsConfiguration(automation)) return AutomationAttention.NeedsConfiguration;
  if (automation.token_revoked_at) return AutomationAttention.TokenRevoked;
  if (automation.pending_version != null) return AutomationAttention.PendingVersion;
  return null;
};

// One rule for the summary, the filter, the sort and the status mark: off wins, then a failed last run, then anything waiting on someone.
export const automationState = (automation: Automation, offlineHostIds: ReadonlySet<string>): AutomationState => {
  if (!automation.enabled) return AutomationState.Off;
  if (automationLastRunFailed(automation)) return AutomationState.Failing;
  if (automationAttention(automation, offlineHostIds)) return AutomationState.Attention;
  return AutomationState.On;
};

export const automationLastRunFailed = (automation: Automation): boolean =>
  automation.last_run != null && automation.last_run.outcome !== AutomationRunOutcome.Success;

// Returned once at mint time (create + rotate) and never again.
export interface AutomationTokenMint {
  automation: Automation;
  token: string;
}

// Semantic pickers ride JSON Schema's `format` ("status"/"channel"/"project") over a base string type.
export type AutomationConfigFieldType = "string" | "number" | "boolean" | "status" | "channel" | "project";

export interface AutomationConfigField {
  key: string;
  label: string;
  type: AutomationConfigFieldType;
  default?: string;
  required?: boolean;
  description?: string;
  // Only for type "status": which project's status columns this picks from; falls back to plain text if absent.
  project_id?: string;
}

export interface AutomationConfigSchema {
  fields: AutomationConfigField[];
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const isScalar = (value: unknown): value is string | number | boolean =>
  typeof value === "string" || typeof value === "number" || typeof value === "boolean";

const parseConfigFieldType = (prop: Record<string, unknown>): AutomationConfigFieldType => {
  if (prop.type === "boolean") return "boolean";
  if (prop.type === "number" || prop.type === "integer") return "number";
  const format = typeof prop.format === "string" ? prop.format : "";
  return format === "status" || format === "channel" || format === "project" ? format : "string";
};

const parseConfigField = (key: string, prop: Record<string, unknown>, required: string[]): AutomationConfigField => ({
  key,
  label: typeof prop.title === "string" && prop.title !== "" ? prop.title : key,
  type: parseConfigFieldType(prop),
  required: required.includes(key),
  ...(isScalar(prop.default) && { default: String(prop.default) }),
  ...(typeof prop.description === "string" && { description: prop.description }),
  ...(typeof prop.project_id === "string" && { project_id: prop.project_id }),
});

// Defensive parse: a malformed schema renders as "no config" instead of crashing the detail page.
export const parseConfigSchema = (raw: unknown): AutomationConfigSchema => {
  if (!isRecord(raw) || !isRecord(raw.properties)) return { fields: [] };
  const required = Array.isArray(raw.required)
    ? raw.required.filter((r): r is string => typeof r === "string")
    : [];
  const fields: AutomationConfigField[] = [];
  for (const [key, prop] of Object.entries(raw.properties)) {
    if (!isRecord(prop)) continue;
    fields.push(parseConfigField(key, prop, required));
  }
  return { fields };
};

// The form holds every value as text; booleans and numbers turn back into JSON types in configValuesToSave.
export const parseConfigValues = (raw: unknown): Record<string, string> => {
  if (!isRecord(raw)) return {};
  return Object.fromEntries(
    Object.entries(raw).flatMap(([key, value]) => (isScalar(value) ? [[key, String(value)]] : [])),
  );
};

const typedConfigValue = (field: AutomationConfigField, value: string): string | number | boolean | undefined => {
  if (field.type === "boolean") return value === "true";
  if (field.type !== "number") return value;
  if (value.trim() === "") return undefined;
  return Number(value);
};

export const configValuesToSave = (
  fields: AutomationConfigField[],
  data: Record<string, string>,
): Record<string, string | number | boolean> =>
  Object.fromEntries(
    fields.flatMap((field) => {
      const value = typedConfigValue(field, data[field.key] ?? "");
      return value === undefined ? [] : [[field.key, value]];
    }),
  );

// The Automation record has no "needs configuration" flag; it's computed client-side from its own schema.
export const automationNeedsConfiguration = (automation: Automation): boolean => {
  const { fields } = parseConfigSchema(automation.config_schema);
  const values = parseConfigValues(automation.config_values);
  return fields.some((f) => f.required && !values[f.key]?.trim());
};

export const CreateAutomationFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  scopes: z.array(z.string()).min(1, "Pick at least one scope"),
});

export type CreateAutomationFormData = z.infer<typeof CreateAutomationFormSchema>;
