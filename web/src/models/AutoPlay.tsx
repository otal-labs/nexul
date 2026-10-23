import type { Play, PlayStage } from "@/models/Play";

// Mirrors internal/plays/autoplay.go: what starts a play by itself, its conditions, priority, and whom it runs on.
export type AutoPlayMoment =
  | "ticket.unblocked"
  | "ticket.entered_stage"
  | "ticket.created"
  | "ticket.developer_set"
  | "ticket.tester_set"
  | "ticket.test_failed"
  | "doc.created"
  | "doc.changed"
  | "stack.service_crashed"
  | "stack.service_recovered"
  | "stack.error_rate_crossed";

export type AutoPlayMatch = "all" | "any";
export type AutoPlayLevel = "high" | "normal" | "low";
export type AutoPlayRunOn = "developer" | "tester" | "causer" | "computer";
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
  | "folder"
  | "stack"
  | "service";

// Values are ids for project, status, folder and stack, names for type, category, label and service, user ids for developer and tester.
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
  // The person who picked a "computer" auto play's computers, whose identity the run takes; "" until someone has (a seeded one) and for every other run-on.
  run_as: string;
  // The computers a "computer" auto play tries in order; empty follows its person's own order. computer_id is the first, for older payloads.
  computer_ids: string[];
  computer_id: string;
  // Read by each run beside the always-included memories; a run reads only those in its own project.
  memory_ids: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

// The full replace the create and update routes take; no computer_ids keeps the stored list and its person.
export type SaveAutoPlayRequest = Pick<
  AutoPlay,
  "enabled" | "moment" | "moment_stage" | "conditions" | "priority" | "once_within_minutes" | "run_on" | "memory_ids"
> & { computer_ids?: string[] };

export const storedComputerIds = (autoPlay: Pick<AutoPlay, "computer_ids" | "computer_id">): string[] =>
  autoPlay.computer_ids ?? (autoPlay.computer_id === "" ? [] : [autoPlay.computer_id]);

// A seeded auto play has no person yet, so nothing has been picked for it and it cannot be switched on.
export const computersPicked = (autoPlay: Pick<AutoPlay, "run_on" | "run_as">) => autoPlay.run_on !== "computer" || autoPlay.run_as !== "";

export interface AutoPlayLimits {
  daily_cap_per_ticket: number;
}

export const MIN_DAILY_CAP = 1;
export const MAX_DAILY_CAP = 50;

export type AutoPlaySubject = "ticket" | "doc" | "stack";

// What an auto play's play fires on; an interview play has no auto plays, so it never gets here.
export const subjectOf = (play: Pick<Play, "type">): AutoPlaySubject => {
  if (play.type === "doc" || play.type === "stack") return play.type;
  return "ticket";
};

export const SUBJECT_MOMENTS: Record<AutoPlaySubject, AutoPlayMoment[]> = {
  ticket: ["ticket.unblocked", "ticket.entered_stage", "ticket.created", "ticket.developer_set", "ticket.tester_set", "ticket.test_failed"],
  doc: ["doc.created", "doc.changed"],
  stack: ["stack.service_crashed", "stack.service_recovered", "stack.error_rate_crossed"],
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
  "stack.service_crashed": "Service crashes",
  "stack.service_recovered": "Service recovers",
  "stack.error_rate_crossed": "Error rate goes over its limit",
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
  "stack.service_crashed": "service crashes",
  "stack.service_recovered": "service recovers",
  "stack.error_rate_crossed": "error rate goes over its limit",
};

export const SUBJECT_FIELDS: Record<AutoPlaySubject, AutoPlayField[]> = {
  ticket: ["type", "project", "stage", "status", "category", "label", "developer", "tester", "source_doc", "linked_pr", "blocked"],
  doc: ["project", "folder"],
  stack: ["project", "stack", "service"],
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
  stack: "Stack",
  service: "Service",
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
  stack: VALUE_OPS,
  service: VALUE_OPS,
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
export const ID_FIELDS: ReadonlySet<AutoPlayField> = new Set(["project", "status", "folder", "stack", "developer", "tester"]);

export const LEVEL_LABELS: Record<AutoPlayLevel, string> = { high: "High", normal: "Normal", low: "Low" };

export const RUN_ON_LABELS: Record<Exclude<AutoPlayRunOn, "computer">, string> = {
  developer: "Developer",
  tester: "Tester",
  causer: "Whoever caused it",
};

// What the Run on select offers: the computer run-on is two choices, the person's own order and a list they picked.
export type RunOnChoice = keyof typeof RUN_ON_LABELS | "own" | "picked";

export const RUN_ON_CHOICE_LABELS: Record<RunOnChoice, string> = { ...RUN_ON_LABELS, own: "Your computers", picked: "Picked computers" };

// Which choices a play's auto plays offer: a doc has no developer or tester, a stack has none of the three.
export const SUBJECT_RUN_ON: Record<AutoPlaySubject, RunOnChoice[]> = {
  ticket: ["developer", "tester", "causer", "own", "picked"],
  doc: ["causer", "own", "picked"],
  stack: ["own", "picked"],
};

const FIRST_RUN_ON: Record<AutoPlaySubject, AutoPlayRunOn> = { ticket: "developer", doc: "causer", stack: "computer" };

type RunOnFields = Pick<AutoPlayDraft, "run_on" | "computer_ids" | "computers_picked">;

// "" for a seeded auto play nobody has picked computers for.
export const runOnChoice = ({ run_on, computer_ids, computers_picked }: RunOnFields): RunOnChoice | "" => {
  if (run_on !== "computer") return run_on;
  if (!computers_picked) return "";
  return computer_ids.length > 0 ? "picked" : "own";
};

// A picked list starts as the viewer's own computers in their order, for them to reorder and trim.
export const draftForChoice = (choice: RunOnChoice, ownIds: string[]): RunOnFields => {
  if (choice === "own") return { run_on: "computer", computer_ids: [], computers_picked: true };
  if (choice === "picked") return { run_on: "computer", computer_ids: ownIds, computers_picked: true };
  return { run_on: choice, computer_ids: [], computers_picked: true };
};

const sameIds = (a: string[], b: string[]) => a.length === b.length && a.every((id, i) => id === b[i]);

// Whose identity a save gives the run: whoever picked the computers before, until the list is changed, then the one saving.
export const runsAsAfterSave = (draft: Pick<AutoPlayDraft, "run_as" | "computer_ids">, saved: string[], viewerId: string) =>
  draft.run_as === "" || !sameIds(draft.computer_ids, saved) ? viewerId : draft.run_as;

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
  // [] follows the person's own order; computers_picked is false until a seeded auto play has been given either.
  computer_ids: string[];
  computers_picked: boolean;
  // Who picked the computers, "" for nobody yet; never sent, it only says who a save leaves the run as.
  run_as: string;
  memory_ids: string[];
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
  run_on: FIRST_RUN_ON[subject],
  computer_ids: [],
  computers_picked: true,
  run_as: "",
  memory_ids: [],
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
  computer_ids: storedComputerIds(autoPlay),
  computers_picked: computersPicked(autoPlay),
  run_as: autoPlay.run_as,
  memory_ids: autoPlay.memory_ids,
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
  ...(draft.run_on === "computer" && draft.computers_picked && { computer_ids: draft.computer_ids }),
  memory_ids: draft.memory_ids,
});

export const toRequest = (autoPlay: AutoPlay): SaveAutoPlayRequest => ({
  enabled: autoPlay.enabled,
  moment: autoPlay.moment,
  moment_stage: autoPlay.moment_stage,
  conditions: autoPlay.conditions,
  priority: autoPlay.priority,
  once_within_minutes: autoPlay.once_within_minutes,
  run_on: autoPlay.run_on,
  ...(autoPlay.run_on === "computer" && computersPicked(autoPlay) && { computer_ids: storedComputerIds(autoPlay) }),
  memory_ids: autoPlay.memory_ids,
});

export const MATCH_OPTIONS = [
  { value: "all", label: "All" },
  { value: "any", label: "Any" },
];

export const LEVEL_OPTIONS = (["high", "normal", "low"] as const).map((value) => ({ value, label: LEVEL_LABELS[value] }));
