import {
  FIELD_OPS,
  LEVEL_LABELS,
  normalRunOn,
  type AutoPlayComputerRule,
  type AutoPlayField,
  type AutoPlayGroup,
  type AutoPlayLevel,
  type AutoPlayMatch,
  type AutoPlayMoment,
  type AutoPlayOp,
  type AutoPlayRule,
  type AutoPlayRunOn,
  type AutoPlaySubject,
} from "@nexul/client-core/autoPlayText";
import type { PlayStage } from "@nexul/client-core/plays";

import { browserTimezone } from "@/models/Meeting";
import type { Play } from "@/models/Play";
import { DEFAULT_SWEEP_RUNS, type AutoPlaySchedule } from "@/models/Schedule";

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
  // Set only for the schedule moment.
  schedule: AutoPlaySchedule | null;
  conditions: { match: AutoPlayMatch; groups: AutoPlayGroup[] };
  priority: AutoPlayPriority;
  once_within_minutes: number;
  run_on: AutoPlayRunOn;
  // The person a "person" auto play runs as, on their computers in their order; "" until someone picks (a seeded one) and for every other run-on.
  run_as: string;
  // Narrows the person's computers; empty or absent leaves every one in.
  computer_rule?: AutoPlayComputerRule;
  // Read by each run beside the always-included memories; a run reads only those in its own project.
  memory_ids: string[];
  // One run at a time: its own, or with a resource, across every auto play naming it.
  one_at_a_time: boolean;
  resource: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// The full replace the create and update routes take; run_as_me makes the caller the person, and leaving it out keeps who it runs as and its computer rule.
export type SaveAutoPlayRequest = Pick<
  AutoPlay,
  "enabled" | "moment" | "moment_stage" | "schedule" | "conditions" | "priority" | "once_within_minutes" | "run_on" | "memory_ids" | "one_at_a_time" | "resource" | "computer_rule"
> & { run_as_me?: boolean };

// What an auto play's play fires on; an interview play has no auto plays, so it never gets here.
export const subjectOf = (play: Pick<Play, "type">): AutoPlaySubject => {
  if (play.type === "doc" || play.type === "stack") return play.type;
  return "ticket";
};

export const SUBJECT_MOMENTS: Record<AutoPlaySubject, AutoPlayMoment[]> = {
  ticket: [
    "ticket.unblocked",
    "ticket.entered_stage",
    "ticket.created",
    "ticket.developer_set",
    "ticket.tester_set",
    "ticket.test_failed",
    "schedule",
  ],
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
  schedule: "On a schedule",
};

export const SUBJECT_FIELDS: Record<AutoPlaySubject, AutoPlayField[]> = {
  ticket: ["type", "project", "stage", "status", "category", "label", "developer", "tester", "source_doc", "linked_pr", "blocked"],
  doc: ["project", "folder"],
  stack: ["project", "stack", "service"],
};

export const takesValues = (op: AutoPlayOp) => op === "is" || op === "is_not";

// Fields whose values are ids, so a value nothing in view names reads as unknown rather than as a raw id.
export const ID_FIELDS: ReadonlySet<AutoPlayField> = new Set(["project", "status", "folder", "stack", "developer", "tester"]);

// How a run-on choice reads for a subject and moment: the person is named by what they did to it.
export const runOnLabel = (runOn: AutoPlayRunOn, subject: AutoPlaySubject, moment: AutoPlayMoment): string => {
  if (runOn === "person") return "A person";
  if (runOn === "developer") return "The ticket's developer";
  if (runOn === "tester") return "The ticket's tester";
  if (subject === "ticket") return "Whoever moved the ticket";
  return moment === "doc.created" ? "Whoever created the doc" : "Whoever changed the doc";
};

// Which choices a play's auto plays offer: a doc has no developer or tester, a stack has none of the three.
export const SUBJECT_RUN_ON: Record<AutoPlaySubject, AutoPlayRunOn[]> = {
  ticket: ["developer", "tester", "causer", "person"],
  doc: ["causer", "person"],
  stack: ["person"],
};

const FIRST_RUN_ON: Record<AutoPlaySubject, AutoPlayRunOn> = { ticket: "developer", doc: "causer", stack: "person" };

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
  cron: string;
  timezone: string;
  // "" for no stop time.
  stop_at: string;
  max_runs: number;
  one_at_a_time: boolean;
  resource: string;
  run_on: AutoPlayRunOn;
  // Who a person auto play runs as, "" for nobody yet; never sent, run_as_me is.
  run_as: string;
  // Makes the caller the person on save; a new auto play starts with it set, as the server makes its creator the person.
  run_as_me: boolean;
  computer_rule: AutoPlayComputerRule;
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
  once_within_minutes: 60,
  run_on: FIRST_RUN_ON[subject],
  cron: "0 23 * * *",
  timezone: browserTimezone(),
  stop_at: "07:00",
  max_runs: DEFAULT_SWEEP_RUNS,
  one_at_a_time: false,
  resource: "",
  run_as: "",
  run_as_me: true,
  computer_rule: {},
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
  cron: autoPlay.schedule?.cron ?? "0 23 * * *",
  timezone: autoPlay.schedule?.timezone ?? browserTimezone(),
  stop_at: autoPlay.schedule?.stop_at ?? "",
  max_runs: autoPlay.schedule?.max_runs ?? DEFAULT_SWEEP_RUNS,
  one_at_a_time: autoPlay.one_at_a_time,
  resource: autoPlay.resource,
  run_on: normalRunOn(autoPlay.run_on),
  run_as: autoPlay.run_as,
  run_as_me: false,
  computer_rule: autoPlay.computer_rule ?? {},
  memory_ids: autoPlay.memory_ids,
});

// Values the editor cannot name (a project the viewer can't open) ride along untouched, so a save never drops them.
export const toSaveRequest = (draft: AutoPlayDraft): SaveAutoPlayRequest => ({
  enabled: draft.enabled,
  moment: draft.moment,
  moment_stage: draft.moment === "ticket.entered_stage" ? draft.moment_stage : null,
  schedule:
    draft.moment === "schedule"
      ? { cron: draft.cron.trim(), timezone: draft.timezone.trim(), stop_at: draft.stop_at, max_runs: draft.max_runs }
      : null,
  conditions: {
    match: draft.match,
    groups: draft.groups.filter((group) => group.rules.length > 0).map(({ match, rules }) => ({ match, rules })),
  },
  priority: draft.priority,
  once_within_minutes: draft.once_within_minutes,
  run_on: draft.run_on,
  ...(draft.run_on === "person" && draft.run_as_me && { run_as_me: true }),
  // Computer ids name computers in a person's order, so only a person auto play keeps them.
  computer_rule: { labels_in: draft.computer_rule.labels_in ?? [], labels_out: draft.computer_rule.labels_out ?? [], ids: draft.run_on === "person" ? (draft.computer_rule.ids ?? []) : [] },
  memory_ids: draft.memory_ids,
  one_at_a_time: draft.one_at_a_time,
  resource: draft.one_at_a_time ? draft.resource.trim() : "",
});

export const toRequest = (autoPlay: AutoPlay): SaveAutoPlayRequest => ({
  enabled: autoPlay.enabled,
  moment: autoPlay.moment,
  moment_stage: autoPlay.moment_stage,
  schedule: autoPlay.schedule,
  conditions: autoPlay.conditions,
  priority: autoPlay.priority,
  once_within_minutes: autoPlay.once_within_minutes,
  run_on: normalRunOn(autoPlay.run_on),
  memory_ids: autoPlay.memory_ids,
  one_at_a_time: autoPlay.one_at_a_time,
  resource: autoPlay.resource,
});

export const MATCH_OPTIONS = [
  { value: "all", label: "All" },
  { value: "any", label: "Any" },
];

export const LEVEL_OPTIONS = (["high", "normal", "low"] as const).map((value) => ({ value, label: LEVEL_LABELS[value] }));
