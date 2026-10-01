import { EveryProject, type TeamMembership, type TeamPerson, type TeamWorkspace } from "@/models/Team";

// What the Team dialog holds for one workspace until Confirm; an unset field keeps the stored value.
export interface WorkspaceDraft {
  added?: boolean;
  removed?: boolean;
  roleId?: string;
  everyProject?: EveryProject;
  projects?: Record<string, string[]>;
  allow?: string[];
  deny?: string[];
}

export type MemberDraft = Record<string, WorkspaceDraft>;

export interface MemberStep {
  kind: "add" | "update" | "remove";
  workspaceId: string;
  workspaceName: string;
  body: Record<string, unknown>;
}

export type MemberDraftAction =
  | { type: "change"; workspaceId: string; change: Pick<WorkspaceDraft, "roleId" | "everyProject" | "allow" | "deny"> }
  | { type: "project"; workspaceId: string; projectId: string; allow: string[] }
  | { type: "add"; workspaceId: string; roleId: string }
  | { type: "remove"; workspaceId: string; removed: boolean }
  | { type: "applied"; step: MemberStep }
  | { type: "reset" };

const without = (draft: MemberDraft, workspaceId: string): MemberDraft =>
  Object.fromEntries(Object.entries(draft).filter(([id]) => id !== workspaceId));

const appliedTo = (draft: MemberDraft, step: MemberStep): MemberDraft => {
  const entry = draft[step.workspaceId];
  if (!entry || step.kind === "remove") return without(draft, step.workspaceId);
  if (step.kind === "add") return { ...draft, [step.workspaceId]: { ...entry, added: false } };
  return { ...draft, [step.workspaceId]: { added: entry.added ?? false, removed: entry.removed ?? false } };
};

export const memberDraftReducer = (draft: MemberDraft, action: MemberDraftAction): MemberDraft => {
  if (action.type === "reset") return {};
  if (action.type === "applied") return appliedTo(draft, action.step);
  if (action.type === "add") return { ...draft, [action.workspaceId]: { added: true, roleId: action.roleId } };
  const entry = draft[action.workspaceId] ?? {};
  if (action.type === "remove" && entry.added) return without(draft, action.workspaceId);
  if (action.type === "remove") return { ...draft, [action.workspaceId]: { ...entry, removed: action.removed } };
  if (action.type === "project") return { ...draft, [action.workspaceId]: { ...entry, projects: { ...entry.projects, [action.projectId]: action.allow } } };
  return { ...draft, [action.workspaceId]: { ...entry, ...action.change } };
};

const freshMembership = (workspace: TeamWorkspace, roleId: string): TeamMembership => ({
  workspace_id: workspace.id,
  workspace_name: workspace.name,
  role_id: roleId,
  role_name: workspace.roles.find((role) => role.id === roleId)?.name ?? "",
  is_owner: false,
  allow: [],
  deny: [],
  every_project: EveryProject.Role,
  projects: [],
});

// The membership as it will read after Confirm: the stored one, or a fresh one for a workspace added here.
export const draftedMembership = (stored: TeamMembership | undefined, workspace: TeamWorkspace, entry: WorkspaceDraft | undefined): TeamMembership | undefined => {
  const base = stored ?? (entry?.added && entry.roleId ? freshMembership(workspace, entry.roleId) : undefined);
  if (!base || !entry) return base;
  const changed = Object.entries(entry.projects ?? {});
  const kept = base.projects.filter((grant) => !changed.some(([projectId]) => projectId === grant.project_id));
  return {
    ...base,
    role_id: entry.roleId ?? base.role_id,
    every_project: entry.everyProject ?? base.every_project,
    allow: entry.allow ?? base.allow,
    deny: entry.deny ?? base.deny,
    projects: [...kept, ...changed.filter(([, allow]) => allow.length > 0).map(([project_id, allow]) => ({ project_id, allow }))],
  };
};

const sameSet = (a: readonly string[], b: readonly string[]) => a.length === b.length && a.every((value) => b.includes(value));

// The PATCH body for what differs from the stored membership.
const patchOf = (stored: TeamMembership, entry: WorkspaceDraft): Record<string, unknown> => {
  const body: Record<string, unknown> = {};
  if (entry.roleId && entry.roleId !== stored.role_id) body.role_id = entry.roleId;
  if (entry.everyProject && entry.everyProject !== stored.every_project) body.every_project = entry.everyProject;
  const access = Object.entries(entry.projects ?? {})
    .filter(([projectId, allow]) => !sameSet(allow, stored.projects.find((grant) => grant.project_id === projectId)?.allow ?? []))
    .map(([project_id, allow]) => ({ project_id, allow }));
  if (access.length > 0) body.project_access = access;
  const allow = entry.allow ?? stored.allow;
  const deny = entry.deny ?? stored.deny;
  if (!sameSet(allow, stored.allow) || !sameSet(deny, stored.deny)) Object.assign(body, { allow, deny });
  return body;
};

// Confirm's requests in the order they apply: adds, then each workspace's changes in one request, then removals.
export const memberSteps = (person: TeamPerson, workspaces: TeamWorkspace[], draft: MemberDraft): MemberStep[] => {
  const steps = workspaces.flatMap((workspace): MemberStep[] => {
    const entry = draft[workspace.id];
    const stored = person.workspaces.find((membership) => membership.workspace_id === workspace.id);
    const added = !stored && entry?.added && entry.roleId ? freshMembership(workspace, entry.roleId) : undefined;
    const base = stored ?? added;
    if (!entry || !base) return [];
    const step = (kind: MemberStep["kind"], body: Record<string, unknown> = {}): MemberStep => ({ kind, workspaceId: workspace.id, workspaceName: workspace.name, body });
    if (entry.removed && stored) return [step("remove")];
    const patch = patchOf(base, entry);
    const update = Object.keys(patch).length > 0 ? [step("update", patch)] : [];
    if (added) return [step("add", { role_id: added.role_id }), ...update];
    return update;
  });
  const order: MemberStep["kind"][] = ["add", "update", "remove"];
  return order.flatMap((kind) => steps.filter((step) => step.kind === kind));
};

// An allow and a deny on the same permission can't both be saved, so Confirm waits until the overrides agree.
export const hasOverlap = (person: TeamPerson, draft: MemberDraft): boolean =>
  Object.entries(draft).some(([workspaceId, entry]) => {
    const stored = person.workspaces.find((membership) => membership.workspace_id === workspaceId);
    const deny = entry.deny ?? stored?.deny ?? [];
    return (entry.allow ?? stored?.allow ?? []).some((value) => deny.includes(value));
  });
