import type { HeldRole } from "@nexul/client-core/permissions";

import type { Role } from "@/models/Role";
import { levelName, levelOf, type PermissionDomain } from "@/models/PermissionLevel";

// One role on a roles checklist: whether the viewer may give or take it, and if not, why.
export interface RoleChoice {
  role: Role;
  givable: boolean;
  reason: string;
}

export const OWNER_REASON = "The Owner role is never given or taken here.";

const REASONS_SHOWN = 2;

const capitalize = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1);

// What a role holds that the viewer's roles don't, combined; an Owner holds everything (ADR 0088).
const missingFrom = (role: Role, mine: Role[]): string[] => {
  if (mine.some((held) => held.is_owner_role)) return [];
  const held = new Set(mine.flatMap((r) => r.permissions));
  return role.permissions.filter((permission) => !held.has(permission));
};

// "Deploys: Delete and Runners: Write": each area the role reaches further in, at the role's own level there.
const beyondText = (domains: PermissionDomain[], role: Role, missing: string[]): string => {
  const parts = domains.flatMap((domain) => {
    if (domain.levels.some((entry) => missing.includes(entry.value))) return [`${domain.name}: ${levelName(levelOf(domain, role.permissions))}`];
    return domain.extras.filter((extra) => missing.includes(extra.value)).map((extra) => `${domain.name}: ${capitalize(extra.action)}`);
  });
  const shown = parts.slice(0, REASONS_SHOWN);
  const more = parts.length - shown.length;
  if (more > 0) return `${shown.join(", ")} and ${more} more`;
  return shown.join(" and ");
};

// The workspace's roles, the Owner first, each marked with whether the viewer may give it.
export const roleChoices = (roles: Role[], myRoles: readonly HeldRole[], domains: PermissionDomain[]): RoleChoice[] => {
  const mine = roles.filter((role) => myRoles.some((held) => held.id === role.id));
  const ordered = [...roles.filter((role) => role.is_owner_role), ...roles.filter((role) => !role.is_owner_role)];
  return ordered.map((role) => {
    if (role.is_owner_role) return { role, givable: false, reason: OWNER_REASON };
    const missing = missingFrom(role, mine);
    if (missing.length === 0) return { role, givable: true, reason: "" };
    return { role, givable: false, reason: `Holds ${beyondText(domains, role, missing)}, which you don't hold.` };
  });
};
