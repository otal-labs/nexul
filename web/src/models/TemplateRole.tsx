import type { WelcomeStep } from "@nexul/client-core/welcome";

import type { Role } from "@/models/Role";
import type { Template } from "@/models/Template";

// Mirrors internal/roles/template.go: a Template role's body, the JSON each new workspace copies into a role of its own.
interface TemplateRoleBody {
  name: string;
  description: string;
  permissions: string[];
  welcome_steps: WelcomeStep[];
  show_locked_actions: boolean;
}

const readBody = (body: string): Partial<TemplateRoleBody> => {
  try {
    return JSON.parse(body) as Partial<TemplateRoleBody>;
  } catch {
    return {};
  }
};

// A template role read as a role, so the role editor and its summary take it as they take a workspace's; its id is the key.
export const templateRole = (template: Template): Role => {
  const body = readBody(template.body);
  return {
    id: template.key,
    workspace_id: "",
    name: body.name ?? template.name,
    description: body.description ?? "",
    permissions: body.permissions ?? [],
    welcome_steps: body.welcome_steps ?? [],
    show_locked_actions: body.show_locked_actions ?? false,
    is_owner_role: false,
    created_at: "",
    updated_at: template.updated_at ?? "",
  };
};

export const templateRoleBody = (
  name: string,
  description: string,
  permissions: string[],
  welcomeSteps: WelcomeStep[],
  showLockedActions: boolean,
): string =>
  JSON.stringify({ name, description, permissions, welcome_steps: welcomeSteps, show_locked_actions: showLockedActions } satisfies TemplateRoleBody);
