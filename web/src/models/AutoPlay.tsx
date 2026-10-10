import type { PlayStage } from "@/models/Play";

// Mirrors internal/plays/autoplay.go: what starts a play by itself, its conditions, priority, and whom it runs on.
export type AutoPlayMoment =
  | "ticket.unblocked"
  | "ticket.entered_stage"
  | "ticket.created"
  | "ticket.developer_set"
  | "ticket.tester_set"
  | "ticket.test_failed"
  | "doc.created"
  | "doc.changed";

export type AutoPlayMatch = "all" | "any";
export type AutoPlayLevel = "high" | "normal" | "low";
export type AutoPlayRunOn = "developer" | "tester" | "causer";
export type AutoPlayOp = "is" | "is_not" | "set" | "unset";
export type AutoPlayField =
  | "type"
  | "project"
  | "stage"
  | "status"
  | "category"
  | "label"
  | "developer"
  | "tester"
  | "source_doc"
  | "linked_pr"
  | "blocked"
  | "folder";

// Values are ids for project, status and folder, names for type, category and label, user ids for developer and tester.
export interface AutoPlayRule {
  field: AutoPlayField;
  op: AutoPlayOp;
  values?: string[];
}

export interface AutoPlayGroup {
  match: AutoPlayMatch;
  rules: AutoPlayRule[];
}

export interface AutoPlayPriority {
  rules: { level: AutoPlayLevel; when: AutoPlayGroup }[];
  otherwise: AutoPlayLevel;
}

export interface AutoPlay {
  id: string;
  play_id: string;
  workspace_id: string;
  enabled: boolean;
  moment: AutoPlayMoment;
  moment_stage: PlayStage | null;
  conditions: { match: AutoPlayMatch; groups: AutoPlayGroup[] };
  priority: AutoPlayPriority;
  once_within_minutes: number;
  run_on: AutoPlayRunOn;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// The full replace the create and update routes take.
export type SaveAutoPlayRequest = Pick<
  AutoPlay,
  "enabled" | "moment" | "moment_stage" | "conditions" | "priority" | "once_within_minutes" | "run_on"
>;

export interface AutoPlayLimits {
  daily_cap_per_ticket: number;
}

export const MIN_DAILY_CAP = 1;
export const MAX_DAILY_CAP = 50;

export type AutoPlaySubject = "ticket" | "doc";

export const SUBJECT_MOMENTS: Record<AutoPlaySubject, AutoPlayMoment[]> = {
  ticket: ["ticket.unblocked", "ticket.entered_stage", "ticket.created", "ticket.developer_set", "ticket.tester_set", "ticket.test_failed"],
  doc: ["doc.created", "doc.changed"],
};

export const MOMENT_LABELS: Record<AutoPlayMoment, string> = {
  "ticket.unblocked": "Ticket becomes unblocked",
  "ticket.entered_stage": "Ticket enters a stage",
  "ticket.created": "Ticket is created",
  "ticket.developer_set": "Ticket gets a developer",
  "ticket.tester_set": "Ticket gets a tester",
  "ticket.test_failed": "Ticket fails a test",
  "doc.created": "Doc is created",
  "doc.changed": "Doc changes",
};

// The moment as the list row's sentence says it after "When a ticket".
export const MOMENT_PHRASES: Record<AutoPlayMoment, string> = {
  "ticket.unblocked": "becomes unblocked",
  "ticket.entered_stage": "enters",
  "ticket.created": "is created",
  "ticket.developer_set": "gets a developer",
  "ticket.tester_set": "gets a tester",
  "ticket.test_failed": "fails a test",
  "doc.created": "is created",
  "doc.changed": "changes",
};

export const SUBJECT_FIELDS: Record<AutoPlaySubject, AutoPlayField[]> = {
  ticket: ["type", "project", "stage", "status", "category", "label", "developer", "tester", "source_doc", "linked_pr", "blocked"],
  doc: ["project", "folder"],
};

export const FIELD_LABELS: Record<AutoPlayField, string> = {
  type: "Type",
  project: "Project",
  stage: "Stage",
  status: "Status",
  category: "Category",
  label: "Label",
  developer: "Developer",
  tester: "Tester",
  source_doc: "Has source doc",
  linked_pr: "Has linked PR",
  blocked: "Blocked",
  folder: "Folder",
};

const VALUE_OPS: AutoPlayOp[] = ["is", "is_not"];
const PRESENCE_OPS: AutoPlayOp[] = ["set", "unset"];
const ANY_OP: AutoPlayOp[] = ["is", "is_not", "set", "unset"];

export const FIELD_OPS: Record<AutoPlayField, AutoPlayOp[]> = {
  type: VALUE_OPS,
  project: VALUE_OPS,
  stage: VALUE_OPS,
  status: VALUE_OPS,
  folder: VALUE_OPS,
  category: ANY_OP,
  label: ANY_OP,
  developer: ANY_OP,
  tester: ANY_OP,
  source_doc: PRESENCE_OPS,
  linked_pr: PRESENCE_OPS,
  blocked: PRESENCE_OPS,
};

const OP_LABELS: Record<AutoPlayOp, string> = { is: "is", is_not: "is not", set: "is set", unset: "is not set" };

// A yes/no field reads its set and unset as yes and no, with no value box after it.
export const opLabel = (field: AutoPlayField, op: AutoPlayOp): string => {
  if (FIELD_OPS[field] === PRESENCE_OPS) return op === "set" ? "yes" : "no";
  return OP_LABELS[op];
};

export const takesValues = (op: AutoPlayOp) => op === "is" || op === "is_not";

// Fields whose values are ids, so a value nothing in view names reads as unknown rather than as a raw id.
export const ID_FIELDS: ReadonlySet<AutoPlayField> = new Set(["project", "status", "folder", "developer", "tester"]);

export const LEVEL_LABELS: Record<AutoPlayLevel, string> = { high: "High", normal: "Normal", low: "Low" };

export const RUN_ON_LABELS: Record<AutoPlayRunOn, string> = {
  developer: "Developer",
  tester: "Tester",
  causer: "Whoever caused it",
};

export const LIMIT_PRESETS = [0, 60, 24 * 60, 7 * 24 * 60];

const plural = (n: number, unit: string) => `${n} ${unit}${n === 1 ? "" : "s"}`;

export const limitLabel = (minutes: number): string => {
  if (minutes === 0) return "No limit";
  if (minutes === 7 * 24 * 60) return "7 days";
  if (minutes % 60 === 0) return plural(minutes / 60, "hour");
  return plural(minutes, "minute");
};

export const blankRule = (field: AutoPlayField): AutoPlayRule => ({ field, op: FIELD_OPS[field][0] ?? "is", values: [] });

// The composer's working copy: a top-level group of one rule shows as a plain condition row, any other as a nested group.
export interface DraftGroup extends AutoPlayGroup {
  nested: boolean;
}

export interface AutoPlayDraft {
  // Which fields and moments apply; never sent.
  subject: AutoPlaySubject;
  enabled: boolean;
  moment: AutoPlayMoment;
  moment_stage: PlayStage;
  match: AutoPlayMatch;
  groups: DraftGroup[];
  priority: AutoPlayPriority;
  once_within_minutes: number;
  run_on: AutoPlayRunOn;
}

export const newDraft = (subject: AutoPlaySubject): AutoPlayDraft => ({
  subject,
  enabled: false,
  moment: SUBJECT_MOMENTS[subject][0] ?? "ticket.unblocked",
  moment_stage: "progress",
  match: "all",
  groups: [],
  priority: { rules: [], otherwise: "normal" },
  once_within_minutes: 24 * 60,
  run_on: subject === "doc" ? "causer" : "developer",
});

export const toDraft = (autoPlay: AutoPlay, subject: AutoPlaySubject): AutoPlayDraft => ({
  subject,
  enabled: autoPlay.enabled,
  moment: autoPlay.moment,
  moment_stage: autoPlay.moment_stage ?? "progress",
  match: autoPlay.conditions.match,
  groups: autoPlay.conditions.groups.map((group) => ({ ...group, nested: group.rules.length !== 1 })),
  priority: autoPlay.priority,
  once_within_minutes: autoPlay.once_within_minutes,
  run_on: autoPlay.run_on,
});

// Values the editor cannot name (a project the viewer can't open) ride along untouched, so a save never drops them.
export const toSaveRequest = (draft: AutoPlayDraft): SaveAutoPlayRequest => ({
  enabled: draft.enabled,
  moment: draft.moment,
  moment_stage: draft.moment === "ticket.entered_stage" ? draft.moment_stage : null,
  conditions: {
    match: draft.match,
    groups: draft.groups.filter((group) => group.rules.length > 0).map(({ match, rules }) => ({ match, rules })),
  },
  priority: draft.priority,
  once_within_minutes: draft.once_within_minutes,
  run_on: draft.run_on,
});

export const toRequest = (autoPlay: AutoPlay): SaveAutoPlayRequest => ({
  enabled: autoPlay.enabled,
  moment: autoPlay.moment,
  moment_stage: autoPlay.moment_stage,
  conditions: autoPlay.conditions,
  priority: autoPlay.priority,
  once_within_minutes: autoPlay.once_within_minutes,
  run_on: autoPlay.run_on,
});

export const MATCH_OPTIONS = [
  { value: "all", label: "All" },
  { value: "any", label: "Any" },
];

export const LEVEL_OPTIONS = (["high", "normal", "low"] as const).map((value) => ({ value, label: LEVEL_LABELS[value] }));
