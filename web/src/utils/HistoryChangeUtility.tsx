import type { HistoryActor } from "@/models/History";
import { viaLabel } from "@/utils/HistoryVersionUtility";

const FIELD_NAMES: Record<string, string> = {
  due_on: "Due",
  finished_at: "Finished",
  blocked_by: "Blocked by",
  found_in: "Found in",
};

// "due_on" reads "Due", an unlisted field its own name capitalised.
export const fieldName = (field: string): string => FIELD_NAMES[field] ?? field.charAt(0).toUpperCase() + field.slice(1).replaceAll("_", " ");

// A label or a blocker is added or removed one at a time, so the change reads "+ security", not "None → security".
const DELTA_FIELDS = new Set(["labels", "blocked_by"]);

export const deltaSign = (field: string, from: string, to: string): "+" | "−" | undefined => {
  if (!DELTA_FIELDS.has(field)) return undefined;
  if (from === "") return "+";
  if (to === "") return "−";
  return undefined;
};

// What the change came through: a play run, an automation, or a person's agent; empty for a person at the browser.
export const actorVia = (actor: HistoryActor): string => {
  if (actor.play_label) return `via ${actor.play_label}`;
  if (actor.automation_name) return `via ${actor.automation_name}`;
  return actor.kind?.endsWith(":mcp") ? viaLabel("mcp") : "";
};

// "Alice via Implement with AI": the person, then what it came through.
export const actorLabel = (actor: HistoryActor, person: string): string => [person, actorVia(actor)].filter(Boolean).join(" ");

// A YYYY-MM-DD day read in local time, without the year while it is this one: "17 Nov", else "17 Nov 2027".
export const dayLabel = (day: string): string => {
  const [year, month, date] = day.split("-").map(Number);
  if (!year || !month || !date) return day;
  const options: Intl.DateTimeFormatOptions = { day: "numeric", month: "short", ...(year === new Date().getFullYear() ? {} : { year: "numeric" }) };
  return new Date(year, month - 1, date).toLocaleDateString(undefined, options);
};
