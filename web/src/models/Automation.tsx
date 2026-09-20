import { z } from "zod";

import type { AutomationKind } from "@/enums/Automation";

// Name/description/subscriptions/config_schema are code-declared and overwritten by the SDK on every dial-in.
export interface Automation {
  id: string;
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
  created_at: string;
  updated_at: string;
}

// Returned once at mint time (create + rotate) and never again.
export interface AutomationTokenMint {
  automation: Automation;
  token: string;
}

// Semantic pickers ride JSON Schema's `format` ("status"/"channel") over a base string type.
export type AutomationConfigFieldType = "string" | "status" | "channel";

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

const parseConfigFieldType = (prop: Record<string, unknown>): AutomationConfigFieldType => {
  const format = typeof prop.format === "string" ? prop.format : "";
  return format === "status" || format === "channel" ? format : "string";
};

const parseConfigField = (key: string, prop: Record<string, unknown>, required: string[]): AutomationConfigField => ({
  key,
  label: typeof prop.title === "string" && prop.title !== "" ? prop.title : key,
  type: parseConfigFieldType(prop),
  required: required.includes(key),
  ...(typeof prop.default === "string" && { default: prop.default }),
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

export const parseConfigValues = (raw: unknown): Record<string, string> => {
  if (!isRecord(raw)) return {};
  return Object.fromEntries(
    Object.entries(raw).filter((entry): entry is [string, string] => typeof entry[1] === "string"),
  );
};

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
