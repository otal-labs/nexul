import type { TeamMembership, TeamPerson } from "@/models/Team";
import { formatRelativeTime } from "@/utils/TimeUtility";

export const personName = (person: TeamPerson): string => person.display_name || person.name || person.login;

// Every account comes from a sign-in, so no session row means signed out everywhere, not never signed in.
export const presenceText = (person: TeamPerson): string => {
  if (person.online) return "Online";
  if (person.last_seen_at) return `Last seen ${formatRelativeTime(person.last_seen_at)}`;
  return "Signed out";
};

export const membershipIn = (person: TeamPerson, workspaceId: string): TeamMembership | undefined =>
  person.workspaces.find((membership) => membership.workspace_id === workspaceId);
