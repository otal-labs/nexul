// The permission each gated area needs; the sidebar and the route guard read it, native/src/models/Access.tsx mirrors it.
export const AREA_PERMISSION = {
  runners: "runners:read",
  topology: "topology:read",
  automations: "automations:read",
  projects: "projects:read",
  tickets: "tickets:read",
  memories: "memories:read",
  docs: "docs:read",
  stacks: "stacks:read",
  deploys: "deploys:read",
  dns: "dns:read",
  newConversation: "chat:write",
  newProject: "projects:write",
  newDoc: "docs:write",
} as const;

export type Area = keyof typeof AREA_PERMISSION;

// The whole-instance Configuration sections and the permission each opens with, held in any workspace (/me's
// instance_permissions). Team is not here: it opens on accounts:read or members:write.
export const INSTANCE_SECTION_PERMISSION = {
  instance: "instance:read",
  "sign-in": "instance:read",
  connectors: "connectors:read",
  dns: "dns:read",
} as const;

export type InstanceSection = keyof typeof INSTANCE_SECTION_PERMISSION;

// Configuration has no bit of its own: it opens when one of its sections does.
export type RouteArea = Area | "configuration";

// A route declares the area it belongs to as its handle; routes without one are open to every member.
export interface RouteAccess {
  area: RouteArea;
}
