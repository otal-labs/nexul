import type { Area } from "@nexul/client-core/permissions";

// The Settings page's instance sections and the permission each opens with, held in any workspace (/me's
// instance_permissions). Team is not here: it opens on accounts:read or members:write.
export const INSTANCE_SECTION_PERMISSION = {
  instance: "instance:read",
  "sign-in": "instance:read",
  connectors: "connectors:read",
  dns: "dns:read",
  templates: "templates:write",
} as const;

export type InstanceSection = keyof typeof INSTANCE_SECTION_PERMISSION;

// Configuration has no bit of its own: it opens when one of its workspace sections does.
export type RouteArea = Area | "configuration";

// A route declares the area it belongs to as its handle; routes without one are open to every member.
export interface RouteAccess {
  area: RouteArea;
}
