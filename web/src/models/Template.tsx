import { z } from "zod";

import type { InterviewQuestion } from "@/models/InterviewTemplate";

// Mirrors internal/templates/model.go (ADR 0103): a template resolves code default, then instance, then workspace or project.
// These are the text kinds, grouped on the Templates card; template roles have a card of their own.
export const TEMPLATE_KINDS = ["interview", "mention_chip", "play_instructions", "ticket_body", "agent_prompt", "memory"] as const;
export type TextTemplateKind = (typeof TEMPLATE_KINDS)[number];

export const ROLE_TEMPLATE = "role";
export type TemplateKind = TextTemplateKind | typeof ROLE_TEMPLATE;

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
  // The instance added it rather than Nexul shipping it: it deletes instead of resetting.
  added: boolean;
  // The body parsed into questions, for the interview kind only.
  questions?: InterviewQuestion[];
  updated_by?: string;
  updated_at?: string;
}

export const INSTANCE: TemplateLocation = { scope: "instance" };

// The instance's Templates groups, each with how its kinds reach workspaces and projects; single templates that
// reach them the same way share a group.
export const TEMPLATE_GROUPS: { label: string; hint: string; kinds: TextTemplateKind[] }[] = [
  { label: "Workspace defaults", hint: "Every workspace follows these until it edits its own.", kinds: ["interview", "mention_chip"] },
  { label: "Play instructions", hint: "Copied into each workspace when it is created.", kinds: ["play_instructions"] },
  { label: "Ticket bodies", hint: "Copied into each project when it is created.", kinds: ["ticket_body"] },
  { label: "Agent prompt", hint: "Opens and closes every Agent prompt.", kinds: ["agent_prompt"] },
  { label: "Memories", hint: "Copied into each project when it is created.", kinds: ["memory"] },
];

// The layer under the instance, or null when the kind lives only at the instance and can never be cloned below it.
export const BELOW_SCOPE: Record<TemplateKind, "workspace" | "project" | null> = {
  interview: "workspace",
  mention_chip: "workspace",
  play_instructions: "workspace",
  ticket_body: "project",
  agent_prompt: null,
  memory: null,
  role: null,
};

// The permission writing a template of each kind takes.
export const WRITE_PERMISSION: Record<TemplateKind, string> = {
  interview: "memories:template",
  mention_chip: "workspaces:write",
  play_instructions: "plays:write",
  ticket_body: "projects:write",
  agent_prompt: "templates:write",
  memory: "templates:write",
  role: "templates:write",
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

// Default memories are written as a SKILL.md; the instance adds its own (ADR 0194), and a new one starts from this.
export const MEMORY_TEMPLATE = "memory";
export const MEMORY_TEMPLATE_SKELETON = `---
name: New memory
description: Use when …
always_included: false
footer: false
---

`;
