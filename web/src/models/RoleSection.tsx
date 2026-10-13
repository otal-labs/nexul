import type { WelcomeStep } from "@nexul/client-core/welcome";

import { PermissionArea, type PermissionInfo } from "@/models/Permission";
import { domainsIn, levelName, levelOf, type PermissionDomain } from "@/models/PermissionLevel";

// The role editor's tabs, widest first; each holds one catalog area.
export interface RoleSection {
  area: PermissionArea;
  title: string;
  bulk: string;
}

export const ROLE_SECTIONS: RoleSection[] = [
  { area: PermissionArea.Instance, title: "Instance", bulk: "All instance areas" },
  { area: PermissionArea.Workspace, title: "Workspace", bulk: "All workspace areas" },
  { area: PermissionArea.Project, title: "In projects", bulk: "All areas in projects" },
];

// The tab after the permission areas: which Welcome wizard steps the role's holders are asked to do.
export const WELCOME_TAB = "welcome";

export type RoleTab = PermissionArea | typeof WELCOME_TAB;

// What a holder is asked to do for each step, the line under its switch.
export const WELCOME_STEP_LINES: Record<WelcomeStep, string> = {
  github: "Link their GitHub account, so repositories open as them.",
  computer: "Install Nexul on their computer and pair T3 Code, so agents can run there.",
  projects: "Open the projects they're in and see what they can do there.",
  deploy_server: "Install Nexul on a server, so stacks can deploy to it.",
  phone: "Sign in to the phone app by scanning a QR code.",
};

export const NO_STEPS: WelcomeStep[] = [];

export const sectionLine =(area: PermissionArea, workspaceName: string): string => {
  if (area === PermissionArea.Instance) return "Applies across every workspace on this install.";
  if (area === PermissionArea.Workspace) return `Across ${workspaceName || "this workspace"}, whatever project someone is in.`;
  return "Only inside the projects a holder is in. Each project lists who's in it.";
};

// A shared empty set: a fresh [] as a fallback is a new value every render and loops whatever compares it.
export const NO_ACTIONS: string[] = [];

// Yes-or-no permissions set with a switch, in the order they lead their tab; the catalog's area picks the tab.
const ABILITY_LINES: Record<string, string> = {
  "projects:create": "A new project lists its creator and the roles they pick.",
  "projects:access": "Choose who's in a project and what they can do there.",
  "memories:template": "Change the questions every interview in this workspace asks.",
};

const ABILITY_ORDER = Object.keys(ABILITY_LINES);

export const abilityLine = (value: string): string | undefined => ABILITY_LINES[value];

export interface SectionDomains {
  // Abilities lead the tab as switches, as does any verb of a domain with no read/write/delete ladder here.
  abilities: PermissionInfo[];
  areas: PermissionDomain[];
}

export const sectionDomains = (catalog: PermissionInfo[], area: PermissionArea): SectionDomains => {
  const named = catalog
    .filter((entry) => entry.area === area && entry.value in ABILITY_LINES)
    .sort((a, b) => ABILITY_ORDER.indexOf(a.value) - ABILITY_ORDER.indexOf(b.value));
  const domains = domainsIn(
    catalog.filter((entry) => !(entry.value in ABILITY_LINES)),
    [area],
  );
  return {
    abilities: [...named, ...domains.filter((domain) => domain.levels.length === 0).flatMap((domain) => domain.extras)],
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

// Plays' first rung is seeing them: each play carries its own See.
const READ_NAME: Record<string, string> = { plays: "See" };

export const readName = (domain: string): string => READ_NAME[domain] ?? "Read";

export const rungName = (domain: string, rung: number): string => (rung === 1 ? readName(domain) : levelName(rung));

// The short word beside an extra verb's checkbox; its full label is the tooltip.
export const extraName = (action: string): string => SHORT[action] ?? action.charAt(0).toUpperCase() + action.slice(1);
