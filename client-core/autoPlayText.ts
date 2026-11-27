import { PLAY_STAGE_LABELS, type PlayStage } from "@nexul/client-core/plays";

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
  | "stack.error_rate_crossed"
  | "schedule";

export type AutoPlayMatch = "all" | "any";
export type AutoPlayLevel = "high" | "normal" | "low";
export type AutoPlayRunOn = "developer" | "tester" | "causer" | "person";

// Narrows the computers an auto play's runs may land on: one of labels_in when set, none of labels_out, one of ids when set (ids need run_on person).
export interface AutoPlayComputerRule {
  labels_in?: string[];
  labels_out?: string[];
  ids?: string[];
}

// The server still reads "computer" as "person", so an older payload or cache can carry it.
export const normalRunOn = (runOn: AutoPlayRunOn | "computer"): AutoPlayRunOn => (runOn === "computer" ? "person" : runOn);
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

export type AutoPlaySubject = "ticket" | "doc" | "stack";

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
  schedule: "comes up on a schedule",
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

// A yes/no rule as a phrase, "Has a linked PR", where the editor's select reads "Has linked PR yes".
export const PRESENCE_PHRASES: Partial<Record<AutoPlayField, Record<"set" | "unset", string>>> = {
  source_doc: { set: "Has a source doc", unset: "Has no source doc" },
  linked_pr: { set: "Has a linked PR", unset: "Has no linked PR" },
  blocked: { set: "Is blocked", unset: "Is not blocked" },
};

export const LEVEL_LABELS: Record<AutoPlayLevel, string> = { high: "High", normal: "Normal", low: "Low" };

// Where a rule runs, read as a place beside named computers rather than a role.
const RUN_ON_PHRASES: Record<Exclude<AutoPlayRunOn, "person">, string> = {
  developer: "The developer's computers",
  tester: "The tester's computers",
  causer: "The computers of whoever caused it",
};

// owner is the person's name as a possessive ("Your", "Sam's"), "" while nobody has been picked.
export const runOnPhrase = (runOn: AutoPlayRunOn | "computer", owner: string): string => {
  const normal = normalRunOn(runOn);
  if (normal !== "person") return RUN_ON_PHRASES[normal];
  return owner === "" ? "Nobody yet" : `${owner} computers`;
};

// A value list reads "frontend, backend or infrastructure", so its own "or" is never confused with the one between rules.
export const listed = (values: string[]) => (values.length < 2 ? values.join("") : `${values.slice(0, -1).join(", ")} or ${values.at(-1)}`);

// One rule in words, "Type is Bug or Incident"; label names each value as the viewer reads it.
export const ruleText = (rule: AutoPlayRule, label: (value: string) => string): string => {
  const presence = (rule.op === "set" || rule.op === "unset") && PRESENCE_PHRASES[rule.field]?.[rule.op];
  return presence || [FIELD_LABELS[rule.field], opLabel(rule.field, rule.op), listed((rule.values ?? []).map(label))].filter(Boolean).join(" ");
};

const DAY_NAMES = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
const DAY_ABBREVIATIONS = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];

const isNumber = (field: string) => /^\d{1,2}$/.test(field);
const twoDigits = (n: string) => n.padStart(2, "0");

// A zone as a person names it: "Europe/London" reads "London", "America/New_York" reads "New York".
export const zoneName = (zone: string) => (zone.split("/").at(-1) ?? zone).replaceAll("_", " ");

const dayNumber = (field: string) => {
  const index = DAY_ABBREVIATIONS.indexOf(field.toLowerCase());
  if (index >= 0) return index;
  return isNumber(field) ? Number(field) % 7 : -1;
};

// The day-of-week field in words: "Every day", "Weekdays", "Weekends", "Mondays and Thursdays"; "" when it is no plain list.
const daysPhrase = (dow: string): string => {
  if (dow === "*") return "Every day";
  if (["1-5", "mon-fri"].includes(dow.toLowerCase())) return "Weekdays";
  const days = dow.split(",").map(dayNumber);
  if (days.some((day) => day < 0)) return "";
  if (days.length === 2 && days.includes(0) && days.includes(6)) return "Weekends";
  const names = days.map((day) => `${DAY_NAMES[day]}s`);
  return names.length === 1 ? (names[0] ?? "") : `${names.slice(0, -1).join(", ")} and ${names.at(-1)}`;
};

// The cron in plain words for the common shapes, "Every day at 23:00, London"; anything else reads as a custom schedule.
export const describeSchedule = (cron: string, zone: string): string => {
  const fields = cron.trim().split(/\s+/);
  if (fields.length !== 5) return "";
  const [minute = "", hour = "", dom = "", month = "", dow = ""] = fields;
  const place = zoneName(zone || "UTC");
  if (!isNumber(minute) || month !== "*") return `A custom schedule, ${place}`;
  if (hour === "*" && dom === "*" && dow === "*") return `Every hour at :${twoDigits(minute)}, ${place}`;
  if (!isNumber(hour)) return `A custom schedule, ${place}`;
  const time = `${twoDigits(hour)}:${twoDigits(minute)}`;
  if (isNumber(dom) && dow === "*") return `On day ${dom} of each month at ${time}, ${place}`;
  const days = dom === "*" ? daysPhrase(dow) : "";
  if (days === "") return `A custom schedule, ${place}`;
  return `${days} at ${time}, ${place}`;
};

// What an auto play's title reads from it.
export interface AutoPlayMomentFields {
  moment: AutoPlayMoment;
  moment_stage: PlayStage | null;
  schedule: { cron: string; timezone?: string } | null;
}

// A schedule's title is the schedule in words; every other moment's is "When a ticket enters In progress".
export const autoPlayTitle = (autoPlay: AutoPlayMomentFields, subject: AutoPlaySubject): string => {
  if (autoPlay.moment === "schedule" && autoPlay.schedule) return describeSchedule(autoPlay.schedule.cron, autoPlay.schedule.timezone ?? "UTC");
  const stage = autoPlay.moment_stage ? ` ${PLAY_STAGE_LABELS[autoPlay.moment_stage]}` : "";
  return `When a ${subject} ${MOMENT_PHRASES[autoPlay.moment]}${stage}`;
};

// The title as a sentence about "it": "When it enters In progress". A stack's moments already read as one.
export const autoPlayLead = (autoPlay: AutoPlayMomentFields, subject: AutoPlaySubject): string => {
  const title = autoPlayTitle(autoPlay, subject);
  if (autoPlay.moment === "schedule" || subject === "stack") return title;
  return title.replace(`When a ${subject} `, "When it ");
};
