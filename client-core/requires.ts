import type { PermissionAction } from "@nexul/client-core/permissionActions";

// "none" is a route no role permission governs: a person's own records, public sign-in, membership alone.
export type RoutePermission = PermissionAction | "none";

// Declares the permission the server checks for the request it wraps; routePermissions.test.ts reads it against the route table.
export const requires = <T>(permission: RoutePermission, request: T): T => {
  void permission;
  return request;
};
