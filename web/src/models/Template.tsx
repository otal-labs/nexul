import { z } from "zod";

// Mirrors internal/templates/model.go (ADR 0103): a template resolves code default, then instance, then workspace or project.
export const TEMPLATE_KINDS = ["interview", "mention_chip", "play_instructions", "ticket_body", "agent_prompt"] as const;
export type TemplateKind = (typeof TEMPLATE_KINDS)[number];

export type TemplateScope = "instance" | "workspace" | "project";

export interface TemplateLocation {
  scope: TemplateScope;
  workspace_id?: string;
  project_id?: string;
}

export interface Template {
  kind: TemplateKind;
  key: string;
  name: string;
  scope: TemplateScope;
  workspace_id?: string;
  project_id?: string;
  body: string;
  // What a reset here gives: the code default at the instance, the instance's text below it.
  default_body: string;
  edited: boolean;
  // The workspace layer follows the instance live until edited (interview, chip) rather than copying it at creation.
  follows: boolean;
  updated_by?: string;
  updated_at?: string;
}

export const INSTANCE: TemplateLocation = { scope: "instance" };

export const TEMPLATE_GROUP_LABELS: Record<TemplateKind, string> = {
  interview: "Interview",
  mention_chip: "Mention chip",
  play_instructions: "Play instructions",
  ticket_body: "Ticket bodies",
  agent_prompt: "Agent prompt",
};

// The layer under the instance, or null when the kind lives only at the instance and can never be cloned below it.
export const BELOW_SCOPE: Record<TemplateKind, "workspace" | "project" | null> = {
  interview: "workspace",
  mention_chip: "workspace",
  play_instructions: "workspace",
  ticket_body: "project",
  agent_prompt: null,
};

// The permission writing a template of each kind takes.
export const WRITE_PERMISSION: Record<TemplateKind, string> = {
  interview: "memories:write",
  mention_chip: "workspaces:write",
  play_instructions: "plays:write",
  ticket_body: "projects:write",
  agent_prompt: "templates:write",
};

// Keys match ignoring case, the way the server matches a project's "Bug" type to the instance's "bug".
export const findTemplate = (templates: Template[] | undefined, kind: TemplateKind, key: string): Template | undefined =>
  templates?.find((t) => t.kind === kind && t.key.toLowerCase() === key.trim().toLowerCase());

// Where a clone reads from: the kind and key name the template, from the place it lives.
export interface CloneSource {
  kind: TemplateKind;
  key: string;
  name: string;
  from: TemplateLocation;
}

// target is the workspace or project id, empty when cloning to the instance.
export const CloneTemplateFormSchema = z
  .object({ scope: z.string().min(1, "Pick where to clone it"), target: z.string() })
  .refine((data) => data.scope === "instance" || data.target !== "", { message: "Pick one", path: ["target"] });

export type CloneTemplateFormData = z.infer<typeof CloneTemplateFormSchema>;

export const cloneTarget = ({ scope, target }: CloneTemplateFormData): TemplateLocation => {
  if (scope === "workspace") return { scope, workspace_id: target };
  if (scope === "project") return { scope, project_id: target };
  return INSTANCE;
};
