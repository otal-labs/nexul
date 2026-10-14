import { hasPermission, projectPermissions, type MyWorkspaceInfo } from "@nexul/client-core/permissions";
import { INSTANCE_PERMISSION_ACTIONS, PERMISSION_LABELS, type PermissionAction } from "@nexul/client-core/permissionActions";

// An item's own answer from GET /api/permissions/explain.
export interface EntityAnswer {
  allowed: boolean;
  source: { sentence: string };
}

// GET /api/permissions/explain: one answer per action the item's rules may set.
export interface EntityExplanation {
  effective: readonly (EntityAnswer & { action: string })[];
}

// The item's answer for permission, or undefined when its rules don't set that action and the role-wide answer stands.
export const answerFor = (explanation: EntityExplanation | undefined, permission: PermissionAction): EntityAnswer | undefined =>
  explanation?.effective.find((answer) => answer.action === permission);

// A machine answers from the actions its card says the viewer holds (ADR 0151), not the role-wide union.
export const machineAnswer = (machine: { actions: readonly string[] }, permission: PermissionAction): EntityAnswer => ({
  allowed: hasPermission(machine.actions, permission),
  source: { sentence: "" },
});

export interface ActionGate {
  permission: PermissionAction;
  projectId?: string | undefined;
  answer?: EntityAnswer | undefined;
  // Reads "Your role can't <verb>" in place of the catalog's label when that label is clumsy, such as "deploy".
  verb?: string | undefined;
}

// What the auth /me says about the instance, which holds the instance-area actions.
export interface InstanceAccess {
  instance_permissions: readonly string[];
  instance_owner: boolean;
}

export type ActionState = { state: "live" } | { state: "hidden" } | { state: "locked"; reason: string };

const LIVE: ActionState = { state: "live" };
const HIDDEN: ActionState = { state: "hidden" };

// "Delete roles" reads "Your role can't delete roles": the catalog's label, first letter lowered.
export const cannotReason = (permission: PermissionAction, verb?: string | undefined): string => {
  if (verb) return `Your role can't ${verb}`;
  const label = PERMISSION_LABELS[permission];
  return `Your role can't ${label.charAt(0).toLowerCase()}${label.slice(1)}`;
};

// Locked needs a role with Show locked actions on (ADR 0152); otherwise an action the viewer can't take is hidden.
export const actionState = ({
  permission,
  projectId,
  answer,
  verb,
  me,
  instance,
}: ActionGate & { me: MyWorkspaceInfo | undefined; instance?: InstanceAccess | undefined }): ActionState => {
  if (!me) return HIDDEN;
  if (answer) {
    if (answer.allowed) return LIVE;
    return lockedOrHidden(me, answer.source.sentence || cannotReason(permission, verb));
  }
  if (INSTANCE_PERMISSION_ACTIONS.includes(permission)) {
    if (!instance) return HIDDEN;
    if (instance.instance_owner || hasPermission(instance.instance_permissions, permission)) return LIVE;
    return lockedOrHidden(me, cannotReason(permission, verb));
  }
  if (me.roles?.some((role) => role.is_owner)) return LIVE;
  if (hasPermission(projectPermissions(me, projectId), permission)) return LIVE;
  return lockedOrHidden(me, cannotReason(permission, verb));
};

const lockedOrHidden = (me: MyWorkspaceInfo, reason: string): ActionState =>
  me.show_locked_actions === true ? { state: "locked", reason } : HIDDEN;
