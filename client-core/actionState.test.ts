// @vitest-environment node
import { describe, expect, it } from "vitest";

import { actionState, machineAnswer, type ActionState, type EntityAnswer, type InstanceAccess } from "@nexul/client-core/actionState";
import type { PermissionAction } from "@nexul/client-core/permissionActions";
import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

const developer = (overrides: Partial<MyWorkspaceInfo> = {}): MyWorkspaceInfo => ({
  role_name: "Junior Developer",
  roles: [{ id: "role-dev", name: "Junior Developer", is_owner: false }],
  permissions: ["roles:read", "chat:write"],
  projects: [
    { project_id: "p-web", actions: ["tickets:write", "deploys:write"] },
    { project_id: "p-api", actions: ["tickets:read"] },
  ],
  show_locked_actions: true,
  ...overrides,
});

const owner: MyWorkspaceInfo = {
  role_name: "Owner",
  roles: [{ id: "role-owner", name: "Owner", is_owner: true }],
  permissions: [],
  show_locked_actions: false,
};

// What /me looked like before the switch existed.
const beforeTheSwitch: MyWorkspaceInfo = { role_name: "Junior Developer", permissions: ["chat:write"] };

const instanceHolder: InstanceAccess = { instance_permissions: ["dns:write"], instance_owner: false };

const answer = (allowed: boolean, sentence: string, kind?: string): EntityAnswer => ({ allowed, source: kind ? { sentence, kind } : { sentence } });

interface Row {
  name: string;
  me: MyWorkspaceInfo | undefined;
  permission: PermissionAction | readonly PermissionAction[];
  projectId?: string;
  entity?: EntityAnswer;
  verb?: string;
  instance?: InstanceAccess;
  expected: ActionState;
}

const rows: Row[] = [
  { name: "a held workspace action is live", me: developer(), permission: "chat:write", expected: { state: "live" } },
  {
    name: "a missing action is locked with the role's reason when a role shows locked actions",
    me: developer(),
    permission: "roles:delete",
    expected: { state: "locked", reason: "Your role can't delete roles" },
  },
  {
    name: "a missing action is hidden when no role shows locked actions",
    me: developer({ show_locked_actions: false }),
    permission: "roles:delete",
    expected: { state: "hidden" },
  },
  { name: "a missing action is hidden for a /me from before the switch", me: beforeTheSwitch, permission: "roles:delete", expected: { state: "hidden" } },
  {
    name: "the reason keeps an acronym's case",
    me: developer(),
    permission: "dns:delete",
    instance: instanceHolder,
    expected: { state: "locked", reason: "Your role can't delete DNS and gateways" },
  },
  { name: "nothing shows while permissions load", me: undefined, permission: "chat:write", expected: { state: "hidden" } },
  { name: "an instance action held is live before any workspace's permissions arrive", me: undefined, permission: "dns:write", instance: instanceHolder, expected: { state: "live" } },
  { name: "an instance action not held stays hidden while permissions load", me: undefined, permission: "dns:delete", instance: instanceHolder, expected: { state: "hidden" } },
  { name: "the Owner is live with nothing listed", me: owner, permission: "stacks:delete", expected: { state: "live" } },
  { name: "a project action held in this project is live", me: developer(), permission: "deploys:write", projectId: "p-web", expected: { state: "live" } },
  {
    name: "a project action held only in another project is locked",
    me: developer(),
    permission: "deploys:write",
    projectId: "p-api",
    expected: { state: "locked", reason: "Your role can't create and update deploys" },
  },
  {
    name: "a rule that refuses what the role allows locks with the rule's sentence",
    me: developer(),
    permission: "chat:write",
    entity: answer(false, "Denied on stack api for bob"),
    expected: { state: "locked", reason: "Denied on stack api for bob" },
  },
  {
    name: "a rule that refuses is hidden when no role shows locked actions",
    me: developer({ show_locked_actions: false }),
    permission: "chat:write",
    entity: answer(false, "Denied on stack api for bob"),
    expected: { state: "hidden" },
  },
  {
    name: "an item refused only because the role doesn't grant it reads as the role's reason",
    me: developer(),
    permission: "deploys:write",
    verb: "deploy",
    entity: answer(false, "Not allowed: role Junior Developer doesn't grant it", "role"),
    expected: { state: "locked", reason: "Your role can't deploy" },
  },
  {
    name: "a refusal without a sentence falls back to the role's reason",
    me: developer(),
    permission: "roles:delete",
    entity: answer(false, ""),
    expected: { state: "locked", reason: "Your role can't delete roles" },
  },
  {
    name: "a rule that allows what the role lacks is live",
    me: developer(),
    permission: "roles:delete",
    entity: answer(true, "Allowed on role Client for bob"),
    expected: { state: "live" },
  },
  {
    name: "a rule can refuse the Owner, who has no locked actions",
    me: owner,
    permission: "stacks:delete",
    entity: answer(false, "Denied on computer laptop for lena"),
    expected: { state: "hidden" },
  },
  {
    name: "a verb replaces the catalog's label in the reason",
    me: developer(),
    permission: "deploys:write",
    projectId: "p-api",
    verb: "deploy",
    expected: { state: "locked", reason: "Your role can't deploy" },
  },
  { name: "an instance action the instance permissions hold is live", me: developer(), permission: "dns:write", instance: instanceHolder, expected: { state: "live" } },
  {
    name: "an instance action is read from the instance permissions, not the workspace's",
    me: developer({ permissions: ["dns:write"] }),
    permission: "dns:write",
    instance: { instance_permissions: [], instance_owner: false },
    expected: { state: "locked", reason: "Your role can't create and update DNS and gateways" },
  },
  {
    name: "an instance action the instance permissions lack is locked under the switch",
    me: developer(),
    permission: "dns:delete",
    instance: instanceHolder,
    expected: { state: "locked", reason: "Your role can't delete DNS and gateways" },
  },
  {
    name: "an instance action the instance permissions lack is hidden without the switch",
    me: developer({ show_locked_actions: false }),
    permission: "dns:delete",
    instance: instanceHolder,
    expected: { state: "hidden" },
  },
  {
    name: "the instance Owner is live for every instance action",
    me: developer(),
    permission: "accounts:delete",
    instance: { instance_permissions: [], instance_owner: true },
    expected: { state: "live" },
  },
  {
    name: "a machine that lists Manage is live whatever the instance permissions say",
    me: developer(),
    permission: "machines:write",
    entity: machineAnswer({ actions: ["machines:see", "machines:write"] }, "machines:write"),
    instance: { instance_permissions: [], instance_owner: false },
    expected: { state: "live" },
  },
  {
    name: "a machine the viewer only sees locks Manage and Delete with the role's reason",
    me: developer(),
    permission: "machines:delete",
    entity: machineAnswer({ actions: ["machines:see"] }, "machines:delete"),
    instance: { instance_permissions: ["machines:delete"], instance_owner: false },
    expected: { state: "locked", reason: "Your role can't delete machines" },
  },
  {
    name: "several permissions are live when the viewer holds each",
    me: developer(),
    permission: ["tickets:write", "deploys:write"],
    projectId: "p-web",
    expected: { state: "live" },
  },
  {
    name: "several permissions lock on the first one the viewer lacks",
    me: developer(),
    permission: ["tickets:write", "stacks:write", "deploys:write"],
    projectId: "p-web",
    expected: { state: "locked", reason: "Your role can't create and update stacks" },
  },
  {
    name: "an allowed answer on the first of several does not make a missing second one live",
    me: developer(),
    permission: ["tickets:write", "stacks:write"],
    projectId: "p-web",
    entity: answer(true, "Allowed on machine prod for bob"),
    expected: { state: "locked", reason: "Your role can't create and update stacks" },
  },
  {
    name: "several permissions hide when the viewer lacks one and no role shows locked actions",
    me: developer({ show_locked_actions: false }),
    permission: ["tickets:write", "stacks:write"],
    projectId: "p-web",
    expected: { state: "hidden" },
  },
  { name: "an instance action waits for the instance permissions to load", me: developer(), permission: "dns:write", expected: { state: "hidden" } },
];

describe("actionState", () => {
  it.each(rows)("$name", ({ me, permission, projectId, entity, verb, instance, expected }) => {
    expect(actionState({ me, permission, projectId, answer: entity, verb, instance })).toEqual(expected);
  });
});
