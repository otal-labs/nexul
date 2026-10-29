import type { HistoryFieldChange } from "@/models/History";
import { PLAY_STAGE_LABELS, type PlayStage } from "@/models/Play";

const FIELD_LABELS: Record<string, string> = {
  enabled: "Enabled",
  show_when_stage: "Shows at",
  excluded_project_ids: "Excluded projects",
};

const ENABLED: Record<string, string> = { true: "On", false: "Off" };

// The server sends a value as stored: "true", a stage key, a comma-joined list of project ids.
const readValue = (field: string, value: string, projectName: (id: string) => string): string => {
  if (field === "enabled") return ENABLED[value] ?? value;
  if (field === "show_when_stage") return PLAY_STAGE_LABELS[value as PlayStage] ?? value;
  if (field === "excluded_project_ids") return value.split(",").filter(Boolean).map(projectName).join(", ");
  return value;
};

// A play's field change worded for the shared field row: "Enabled On → Off", "Shows at Backlog → In progress", projects by name. An empty side stays empty, which the row reads as None.
export const readPlayChange =
  (projectName: (id: string) => string) =>
  (change: HistoryFieldChange): HistoryFieldChange => ({
    ...change,
    field: FIELD_LABELS[change.field] ?? change.field,
    from: readValue(change.field, change.from, projectName),
    to: readValue(change.field, change.to, projectName),
  });
