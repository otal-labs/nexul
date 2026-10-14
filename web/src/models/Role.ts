import type { WelcomeStep } from "@nexul/client-core/welcome";

export interface Role {
  id: string;
  workspace_id: string;
  name: string;
  // One line on what holders do; empty for roles made before descriptions, which show their name alone.
  description: string;
  permissions: string[];
  // The Welcome wizard steps the role gives its holders; the Owner role's are fixed by the server.
  welcome_steps: WelcomeStep[];
  // Whether its holders see an action they can't take dimmed with the reason, instead of not at all.
  show_locked_actions: boolean;
  is_owner_role: boolean;
  created_at: string;
  updated_at: string;
}

// Mirrors roles.DescriptionMax on the server, which refuses a longer description.
export const ROLE_DESCRIPTION_MAX = 140;

// A duplicate's name: "Admin (copy)", then "Admin (copy 2)" and on while those are taken.
export const copyName = (name: string, taken: readonly string[]): string => {
  const base = `${name} (copy)`;
  if (!taken.includes(base)) return base;
  let n = 2;
  while (taken.includes(`${name} (copy ${n})`)) n++;
  return `${name} (copy ${n})`;
};
