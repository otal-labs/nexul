import type { TeamMembership, TeamPerson } from "@/models/Team";

export const personName = (person: TeamPerson): string => person.display_name || person.name || person.login;

// "Owner in Nexul · Member in Acme", the one line the Team list shows per person.
export const accessSummary = (person: TeamPerson): string =>
  person.workspaces.map((membership) => `${membership.role_name} in ${membership.workspace_name}`).join(" · ");

export const membershipIn = (person: TeamPerson, workspaceId: string): TeamMembership | undefined =>
  person.workspaces.find((membership) => membership.workspace_id === workspaceId);
