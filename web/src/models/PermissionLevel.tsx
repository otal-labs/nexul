import type { PermissionArea, PermissionInfo } from "@/models/Permission";

// Read, write, delete stack: each level grants every action before it, so a domain is one choice, not three boxes.
const LEVEL_ACTIONS = ["read", "write", "delete"];

export interface PermissionDomain {
  domain: string;
  name: string;
  levels: PermissionInfo[];
  extras: PermissionInfo[];
}

const capitalize = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1);

// No client-side domain name map: the read entry's own label ("Read docs") minus its verb is the domain name.
export const domainLabel = (entries: PermissionInfo[]): string => {
  const source = entries.find((entry) => entry.action === "read") ?? entries[0];
  if (!source) return "";
  if (source.action !== "read") return capitalize(source.label);
  return capitalize(source.label.replace(/^Read\s+/, ""));
};

// Server sends entries pre-grouped by domain; this only buckets them, it never reorders.
export const groupByDomain = (entries: PermissionInfo[]): [string, PermissionInfo[]][] => {
  const byDomain = new Map<string, PermissionInfo[]>();
  for (const entry of entries) {
    byDomain.set(entry.domain, [...(byDomain.get(entry.domain) ?? []), entry]);
  }
  return [...byDomain];
};

export const domainsOf = (entries: PermissionInfo[]): PermissionDomain[] =>
  groupByDomain(entries).map(([domain, domainEntries]) => ({
    domain,
    name: domainLabel(domainEntries),
    levels: LEVEL_ACTIONS.flatMap((action) => domainEntries.filter((entry) => entry.action === action)),
    extras: domainEntries.filter((entry) => !LEVEL_ACTIONS.includes(entry.action)),
  }));

export const levelName = (level: number): string => (level === 0 ? "None" : capitalize(LEVEL_ACTIONS[level - 1] ?? ""));

// The highest ladder action held, so a stored set that skips a rung (write without read) still shows.
export const levelOf = (domain: PermissionDomain, value: string[]): number =>
  domain.levels.reduce((level, entry, index) => (value.includes(entry.value) ? index + 1 : level), 0);

// None clears the domain's extra verbs too: no access means none.
export const withLevel = (value: string[], domain: PermissionDomain, level: number): string[] => {
  const cleared = level === 0 ? [...domain.levels, ...domain.extras] : domain.levels;
  const owned = new Set(cleared.map((entry) => entry.value));
  const granted = domain.levels.slice(0, level).map((entry) => entry.value);
  return [...value.filter((v) => !owned.has(v)), ...granted];
};

// A domain with a shorter ladder takes its own top rung, so "Delete" on every row leaves read-only domains at Read.
export const withLevelEverywhere = (value: string[], domains: PermissionDomain[], level: number): string[] =>
  domains.reduce((next, domain) => withLevel(next, domain, Math.min(level, domain.levels.length)), value);

export const uniformLevelOf = (domains: PermissionDomain[], value: string[]): number | undefined => {
  const top = Math.max(0, ...domains.map((domain) => domain.levels.length));
  for (let level = 0; level <= top; level++) {
    if (domains.every((domain) => levelOf(domain, value) === Math.min(level, domain.levels.length))) return level;
  }
  return undefined;
};

export const withExtra = (value: string[], extra: PermissionInfo, on: boolean): string[] => {
  const rest = value.filter((v) => v !== extra.value);
  if (!on) return rest;
  return [...rest, extra.value];
};

// One line under each rung in a level menu, so the ladder explains itself where it is picked.
export const LEVEL_HINTS = ["No access", "Open and read", "Create and edit", "Also delete"];

// A domain's level with its extra verbs, the way a level dropdown reads ("Write + Clone").
export const levelLabel = (domain: PermissionDomain, value: string[]): string => {
  const extras = domain.extras.filter((extra) => value.includes(extra.value)).map((extra) => capitalize(extra.action));
  return [levelName(levelOf(domain, value)), ...extras].join(" + ");
};

// The domains of the given areas, in catalog order: the role editor's two sections and a project's areas.
export const domainsIn = (entries: PermissionInfo[], areas: readonly PermissionArea[]): PermissionDomain[] =>
  domainsOf(entries.filter((entry) => areas.includes(entry.area)));

// A project's one dropdown value: its shared level, or Custom once its areas differ.
export const projectLevelLabel = (domains: PermissionDomain[], value: string[]): string => {
  const uniform = uniformLevelOf(domains, value);
  if (uniform === undefined) return "Custom";
  return levelName(uniform);
};

// "tickets Write · docs Read": the areas with any level, or the one shared level; "None" when nothing is held.
export const accessSummary = (domains: PermissionDomain[], value: string[]): string => {
  const uniform = uniformLevelOf(domains, value);
  if (uniform !== undefined && uniform > 0) return `every area ${levelName(uniform)}`;
  const parts = domains
    .filter((domain) => levelOf(domain, value) > 0)
    .map((domain) => `${domain.name.toLowerCase()} ${levelName(levelOf(domain, value))}`);
  if (parts.length === 0) return "None";
  return parts.join(" · ");
};

// What one edit changed, for the toast: "tickets → Write", or "every area → Read" when several moved.
export const changeSummary = (domains: PermissionDomain[], before: string[], after: string[]): string => {
  const changed = domains.filter((domain) => levelLabel(domain, before) !== levelLabel(domain, after));
  const first = changed[0];
  if (!first) return "access updated";
  if (changed.length > 1) return `every area → ${projectLevelLabel(domains, after)}`;
  return `${first.name.toLowerCase()} → ${levelLabel(first, after)}`;
};

export interface LevelTally {
  // Domains per level, None first; a read-only domain at Read counts under Read.
  counts: number[];
  // Domains the set holds nothing in, by name, in catalog order.
  missing: string[];
  // Extra verbs held (Run, Clone, Thread), counted across every domain.
  extras: number;
}

// How many domains sit at each level, for a role that reads at a glance.
export const levelTally = (domains: PermissionDomain[], value: string[]): LevelTally => {
  const counts = Array.from({ length: LEVEL_ACTIONS.length + 1 }, () => 0);
  const missing: string[] = [];
  let extras = 0;
  for (const domain of domains) {
    const level = levelOf(domain, value);
    counts[level] = (counts[level] ?? 0) + 1;
    if (level === 0) missing.push(domain.name);
    extras += domain.extras.filter((extra) => value.includes(extra.value)).length;
  }
  return { counts, missing, extras };
};
