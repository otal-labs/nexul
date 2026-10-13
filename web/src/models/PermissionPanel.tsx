import { RuleState, type PermissionGrant, type PermissionInfo, type PermissionSource, type RuleChange } from "@/models/Permission";
import { domainLabel } from "@/models/PermissionLevel";

// What the permissions panel opens on. A doc or folder names its project, a doc its folder, for the header and the add list.
export interface PanelEntity {
  type: "doc" | "folder" | "play" | "project" | "computer";
  id: string;
  name: string;
  workspaceId: string;
  projectId: string;
  folderId: string;
}

// The project a doc's or folder's panel lists targets of, refusing anyone outside it.
export const targetsProject = (entity: Pick<PanelEntity, "type" | "projectId">) =>
  entity.type === "doc" || entity.type === "folder" ? entity.projectId : undefined;

// A project as the panel opens on it.
export const projectEntity = (project: { id: string; name: string; workspace_id: string }): PanelEntity => ({
  type: "project", id: project.id, name: project.name, workspaceId: project.workspace_id, projectId: project.id, folderId: "",
});

// One of the viewer's computers as the panel opens on it, to share it.
export const computerEntity = (computer: { id: string; name: string }): PanelEntity => ({
  type: "computer", id: computer.id, name: computer.name, workspaceId: "", projectId: "", folderId: "",
});

// Whom the panel's targets come from: the entity's workspace, or for a computer every workspace the viewer is in.
export const targetScope = (entity: PanelEntity, projectId?: string) => ({
  workspaceId: entity.workspaceId,
  projectId,
  everyWorkspace: entity.type === "computer",
});

export interface PanelGroup {
  name: string;
  actions: string[];
}

// A rule's target as one string, so held rules, the column and the store key roles and people alike.
export type TargetKind = "role" | "person";

export const targetKey = (kind: TargetKind, id: string) => `${kind}:${id}`;

export const parseTarget = (key: string): { kind: TargetKind; id: string } => {
  const at = key.indexOf(":");
  return { kind: key.slice(0, at) === "role" ? "role" : "person", id: key.slice(at + 1) };
};

export const grantKey = (grant: PermissionGrant) =>
  grant.target === "role" ? targetKey("role", grant.role_id ?? "") : targetKey("person", grant.user_id);

// The entity-specific names the panel gives a permission; every other one reads as the catalog labels it.
const panelLabels: Record<PanelEntity["type"], Record<string, string>> = {
  doc: { "permissions:write": "Manage this doc's access" },
  folder: { "permissions:write": "Manage this folder's access" },
  play: { "plays:run": "Run" },
  project: { "projects:open": "See this project", "projects:access": "Manage this project's access", "permissions:write": "Manage each doc's access" },
  computer: { "computers:see": "See this computer", "computers:run_agents": "Run agents", "computers:run_commands": "Run commands" },
};

export const permissionLabel = (action: string, catalog: PermissionInfo[] | undefined, type: PanelEntity["type"] = "doc") =>
  panelLabels[type][action] ?? catalog?.find((c) => c.value === action)?.label ?? action;

const fixedGroups: Record<string, string> = { projects: "Project", plays: "Play", permissions: "Access", computers: "Computer" };

const groupOf = (action: string, catalog: PermissionInfo[] | undefined) => {
  const domain = action.slice(0, action.indexOf(":"));
  return fixedGroups[domain] ?? (domainLabel((catalog ?? []).filter((c) => c.domain === domain)) || domain);
};

// The actions the server says the entity's rules may set, grouped by domain in the order they came.
export const groupActions = (actions: string[], catalog: PermissionInfo[] | undefined): PanelGroup[] => {
  const groups: PanelGroup[] = [];
  for (const action of actions) {
    const name = groupOf(action, catalog);
    const group = groups.find((g) => g.name === name);
    if (group) group.actions.push(action);
    if (!group) groups.push({ name, actions: [action] });
  }
  return groups;
};

export const ruleKey = (target: string, action: string) => `${target} ${action}`;

export const savedState = (grant: PermissionGrant | undefined, action: string): RuleState => {
  if (grant?.deny.includes(action)) return RuleState.Deny;
  if (grant?.allow.includes(action)) return RuleState.Allow;
  return RuleState.Fallback;
};

// Every held change that differs from what is saved, one request per target and state.
export const changesOf = (held: Record<string, RuleState>, grants: PermissionGrant[]): RuleChange[] => {
  const changes = new Map<string, RuleChange>();
  for (const [key, state] of Object.entries(held)) {
    const [target = "", action = ""] = key.split(" ");
    if (savedState(grants.find((g) => grantKey(g) === target), action) === state) continue;
    const change = changes.get(`${target} ${state}`) ?? { target, state, actions: [] };
    change.actions.push(action);
    changes.set(`${target} ${state}`, change);
  }
  return [...changes.values()];
};

export const rulesWord = (n: number) => (n === 0 ? "no rules" : `${n} rule${n === 1 ? "" : "s"}`);

// The dot beside a source: allowed, denied by a rule, or simply not given.
export const verdictTone = (allowed: boolean, source: PermissionSource) => {
  if (allowed) return "allowed";
  if (source.kind === "rule") return "denied";
  return "none";
};

// A name in a source sentence: the entity, or the role, which read in medium weight and may link.
export interface SentencePart {
  text: string;
  names?: "entity" | "role";
}

// A sentence split around the entity and the role it names; a person's name stays plain.
export const sentenceParts = (source: PermissionSource): SentencePart[] => {
  const names: { n: string; names: "entity" | "role" }[] = [];
  if (source.entity_type !== "workspace" && source.entity_name) names.push({ n: source.entity_name, names: "entity" });
  if (source.target_type === "role" && source.target_name) names.push({ n: source.target_name, names: "role" });
  const parts: SentencePart[] = [];
  let rest = source.sentence;
  for (;;) {
    const hits = names.map((name) => ({ ...name, at: rest.indexOf(name.n) })).filter((h) => h.at >= 0);
    const first = hits.sort((a, b) => a.at - b.at)[0];
    if (!first) break;
    if (first.at > 0) parts.push({ text: rest.slice(0, first.at) });
    parts.push({ text: first.n, names: first.names });
    rest = rest.slice(first.at + first.n.length);
    names.splice(names.findIndex((name) => name.n === first.n && name.names === first.names), 1);
  }
  if (rest) parts.push({ text: rest });
  return parts;
};
