import type { PermissionGrant } from "@/models/Permission";

// In this project: held only through a project's rules (ADR 0148).
export const IN_THIS_PROJECT = "projects:open";

export interface ProjectPeopleRules {
  roles: PermissionGrant[];
  people: PermissionGrant[];
  keptOut: PermissionGrant[];
}

// A project's rules as its people card lists them, oldest first: the roles it lets in, the people with a rule of their
// own, and the people a rule keeps out whatever their roles.
export const projectPeopleRules = (grants: PermissionGrant[]): ProjectPeopleRules => {
  const byAge = [...grants].sort((a, b) => a.created_at.localeCompare(b.created_at));
  const people = byAge.filter((g) => g.target === "person");
  return {
    roles: byAge.filter((g) => g.target === "role" && g.allow.includes(IN_THIS_PROJECT)),
    people: people.filter((g) => !g.deny.includes(IN_THIS_PROJECT)),
    keptOut: people.filter((g) => g.deny.includes(IN_THIS_PROJECT)),
  };
};

// The rules a target holds here beyond being in or out.
export const rulesHere = (grant: PermissionGrant) => [...grant.allow, ...grant.deny].filter((a) => a !== IN_THIS_PROJECT).length;

export const rulesHereWord = (n: number) => `${n} rule${n === 1 ? "" : "s"} here`;

export const peopleWord = (n: number) => `${n} ${n === 1 ? "person" : "people"}`;

export const holdItWord = (n: number) => `${peopleWord(n)} ${n === 1 ? "holds" : "hold"} it`;
