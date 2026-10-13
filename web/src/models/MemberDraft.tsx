import { EveryProject, heldRoles, type TeamMembership, type TeamPerson, type TeamWorkspace } from "@/models/Team";

// What the Team dialog holds for one workspace until Confirm; an unset field keeps the stored value.
export interface WorkspaceDraft {
  added?: boolean;
  removed?: boolean;
  roleIds?: string[];
}

export type MemberDraft = Record<string, WorkspaceDraft>;

export interface MemberStep {
  kind: "add" | "update" | "remove";
  workspaceId: string;
  workspaceName: string;
  body: Record<string, unknown>;
}

export type MemberDraftAction =
  | { type: "roles"; workspaceId: string; roleIds: string[] }
  | { type: "add"; workspaceId: string }
  | { type: "remove"; workspaceId: string; removed: boolean }
  | { type: "applied"; step: MemberStep }
  | { type: "reset" };

const without = (draft: MemberDraft, workspaceId: string): MemberDraft =>
  Object.fromEntries(Object.entries(draft).filter(([id]) => id !== workspaceId));

const appliedTo = (draft: MemberDraft, step: MemberStep): MemberDraft => {
  const entry = draft[step.workspaceId];
  if (!entry || step.kind !== "update") return without(draft, step.workspaceId);
  return { ...draft, [step.workspaceId]: { removed: entry.removed ?? false } };
};

export const memberDraftReducer = (draft: MemberDraft, action: MemberDraftAction): MemberDraft => {
  if (action.type === "reset") return {};
  if (action.type === "applied") return appliedTo(draft, action.step);
  if (action.type === "add") return { ...draft, [action.workspaceId]: { added: true, roleIds: [] } };
  const entry = draft[action.workspaceId] ?? {};
  if (action.type === "remove" && entry.added) return without(draft, action.workspaceId);
  if (action.type === "remove") return { ...draft, [action.workspaceId]: { ...entry, removed: action.removed } };
  return { ...draft, [action.workspaceId]: { ...entry, roleIds: action.roleIds } };
};

const freshMembership = (workspace: TeamWorkspace): TeamMembership => ({
  workspace_id: workspace.id,
  workspace_name: workspace.name,
  role_id: "",
  role_name: "",
  roles: [],
  is_owner: false,
  allow: [],
  deny: [],
  every_project: EveryProject.Role,
  projects: [],
});

// The membership as it will read after Confirm: the stored one, or a fresh one for a workspace added here.
export const draftedMembership = (stored: TeamMembership | undefined, workspace: TeamWorkspace, entry: WorkspaceDraft | undefined): TeamMembership | undefined => {
  const base = stored ?? (entry?.added ? freshMembership(workspace) : undefined);
  if (!base || !entry?.roleIds) return base;
  const roles = workspace.roles.filter((role) => entry.roleIds?.includes(role.id));
  return { ...base, roles };
};

const sameSet = (a: readonly string[], b: readonly string[]) => a.length === b.length && a.every((value) => b.includes(value));

// Confirm's requests in the order they apply: adds, then each workspace's new role set, then removals.
export const memberSteps = (person: TeamPerson, workspaces: TeamWorkspace[], draft: MemberDraft): MemberStep[] => {
  const steps = workspaces.flatMap((workspace): MemberStep[] => {
    const entry = draft[workspace.id];
    if (!entry) return [];
    const stored = person.workspaces.find((membership) => membership.workspace_id === workspace.id);
    const step = (kind: MemberStep["kind"], body: Record<string, unknown> = {}): MemberStep => ({ kind, workspaceId: workspace.id, workspaceName: workspace.name, body });
    if (!stored && entry.added) return [step("add", { role_ids: entry.roleIds ?? [] })];
    if (!stored) return [];
    if (entry.removed) return [step("remove")];
    const held = heldRoles(stored).map((role) => role.id);
    if (!entry.roleIds || sameSet(entry.roleIds, held)) return [];
    return [step("update", { role_ids: entry.roleIds })];
  });
  const order: MemberStep["kind"][] = ["add", "update", "remove"];
  return order.flatMap((kind) => steps.filter((step) => step.kind === kind));
};

// Joining a workspace takes at least one role, so Confirm waits until every workspace added here has one.
export const hasAddWithoutRoles = (draft: MemberDraft): boolean =>
  Object.values(draft).some((entry) => entry.added && (entry.roleIds ?? []).length === 0);
