import { RuleState, type PermissionGrant, type PermissionInfo, type PermissionSource, type RuleChange } from "@/models/Permission";

// What the permissions panel opens on. A doc names its project and folder for the header and the add list.
export interface PanelEntity {
  type: "doc" | "play";
  id: string;
  name: string;
  workspaceId: string;
  projectId: string;
  folderId: string;
}

export interface PanelGroup {
  name: string;
  actions: string[];
}

// The entity-specific names the panel gives a permission; every other one reads as the catalog labels it.
const panelLabels: Record<string, string> = {
  "permissions:write": "Manage this doc's access",
  "plays:run": "Run",
};

export const permissionLabel = (action: string, catalog: PermissionInfo[] | undefined) =>
  panelLabels[action] ?? catalog?.find((c) => c.value === action)?.label ?? action;

const groupOf = (action: string) => {
  if (action.startsWith("docs:")) return "Docs";
  if (action.startsWith("plays:")) return "Play";
  return "Access";
};

// The actions the server says the entity's rules may set, grouped by domain in the order they came.
export const groupActions = (actions: string[]): PanelGroup[] => {
  const groups: PanelGroup[] = [];
  for (const action of actions) {
    const name = groupOf(action);
    const group = groups.find((g) => g.name === name);
    if (group) group.actions.push(action);
    if (!group) groups.push({ name, actions: [action] });
  }
  return groups;
};

export const ruleKey = (userId: string, action: string) => `${userId} ${action}`;

export const savedState = (grant: PermissionGrant | undefined, action: string): RuleState => {
  if (grant?.deny.includes(action)) return RuleState.Deny;
  if (grant?.allow.includes(action)) return RuleState.Allow;
  return RuleState.Fallback;
};

// Every held change that differs from what is saved, one request per person and state.
export const changesOf = (held: Record<string, RuleState>, grants: PermissionGrant[]): RuleChange[] => {
  const changes = new Map<string, RuleChange>();
  for (const [key, state] of Object.entries(held)) {
    const [userId = "", action = ""] = key.split(" ");
    if (savedState(grants.find((g) => g.user_id === userId), action) === state) continue;
    const change = changes.get(`${userId} ${state}`) ?? { userId, state, actions: [] };
    change.actions.push(action);
    changes.set(`${userId} ${state}`, change);
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

export interface SentencePart {
  text: string;
  strong: boolean;
}

// A sentence split around the entity and the role it names, which read in medium weight; a person's name stays plain.
export const sentenceParts = (source: PermissionSource): SentencePart[] => {
  const names = [source.entity_type !== "workspace" ? source.entity_name : "", source.target_type === "role" ? source.target_name : ""].filter(Boolean);
  const parts: SentencePart[] = [];
  let rest = source.sentence;
  for (;;) {
    const hits = names.map((n) => ({ n, at: rest.indexOf(n) })).filter((h) => h.at >= 0);
    const first = hits.sort((a, b) => a.at - b.at)[0];
    if (!first) break;
    if (first.at > 0) parts.push({ text: rest.slice(0, first.at), strong: false });
    parts.push({ text: first.n, strong: true });
    rest = rest.slice(first.at + first.n.length);
    names.splice(names.indexOf(first.n), 1);
  }
  if (rest) parts.push({ text: rest, strong: false });
  return parts;
};
