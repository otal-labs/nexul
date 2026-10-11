import { PermissionArea, type PermissionInfo } from "@/models/Permission";
import { domainsIn, levelOf, type PermissionDomain } from "@/models/PermissionLevel";

// The role editor's tabs, widest first; each holds one catalog area.
export interface RoleSection {
  area: PermissionArea;
  title: string;
  bulk: string;
}

export const ROLE_SECTIONS: RoleSection[] = [
  { area: PermissionArea.Instance, title: "Instance", bulk: "All instance areas" },
  { area: PermissionArea.Workspace, title: "Workspace", bulk: "All workspace areas" },
  { area: PermissionArea.Project, title: "Every project", bulk: "All project areas" },
];

export const sectionLine = (area: PermissionArea, workspaceName: string): string => {
  if (area === PermissionArea.Instance) return "Applies across every workspace on this install.";
  if (area === PermissionArea.Workspace) return `Across ${workspaceName || "this workspace"}, whatever project someone is in.`;
  return "Applies to members whose Every project is From role.";
};

// A shared empty set: a fresh [] as a fallback is a new value every render and loops whatever compares it.
export const NO_ACTIONS: string[] = [];

export interface SectionDomains {
  // Domains with no read/write/delete ladder render as switches, one per verb.
  abilities: PermissionInfo[];
  areas: PermissionDomain[];
}

export const sectionDomains = (catalog: PermissionInfo[], area: PermissionArea): SectionDomains => {
  const domains = domainsIn(catalog, [area]);
  return {
    abilities: domains.filter((domain) => domain.levels.length === 0).flatMap((domain) => domain.extras),
    areas: domains.filter((domain) => domain.levels.length > 0),
  };
};

// "12 of 24 allowed": areas above None plus abilities switched on.
export const sectionTally = ({ abilities, areas }: SectionDomains, actions: string[]): { allowed: number; total: number } => ({
  allowed: abilities.filter((ability) => actions.includes(ability.value)).length + areas.filter((domain) => levelOf(domain, actions) > 0).length,
  total: abilities.length + areas.length,
});

export const sameActions = (a: string[], b: string[]): boolean => a.length === b.length && a.every((value) => b.includes(value));

// Whether the draft differs from what was loaded inside one area, for the tab's unsaved mark.
export const changedIn = (catalog: PermissionInfo[], area: PermissionArea, actions: string[], loaded: string[]): boolean =>
  catalog.some((entry) => entry.area === area && actions.includes(entry.value) !== loaded.includes(entry.value));

const SHORT: Record<string, string> = { thread: "Threads" };

// The short word beside an extra verb's checkbox; its full label is the tooltip.
export const extraName = (action: string): string => SHORT[action] ?? action.charAt(0).toUpperCase() + action.slice(1);
