// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { HistoryFieldChange } from "@/models/History";
import { readPlayChange } from "@/utils/PlayHistoryUtility";

const change = (field: string, from: string, to: string): HistoryFieldChange => ({ id: "c-1", field, from, to, created_at: "2026-10-29T10:00:00Z", actor: {} });
const names: Record<string, string> = { "p-1": "Acme web", "p-2": "Globex api" };
const read = readPlayChange((id) => names[id] ?? "A removed project");

describe("readPlayChange", () => {
  it.each([
    ["enabled", "true", "false", "Enabled", "On", "Off"],
    ["show_when_stage", "backlog", "progress", "Shows at", "Backlog", "In progress"],
    ["excluded_project_ids", "p-1", "p-1,p-2", "Excluded projects", "Acme web", "Acme web, Globex api"],
    ["excluded_project_ids", "p-1", "", "Excluded projects", "Acme web", ""],
    ["excluded_project_ids", "", "p-9", "Excluded projects", "", "A removed project"],
    ["label", "Fix it", "Fix it fast", "label", "Fix it", "Fix it fast"],
  ])("words a change of %s as the rail shows it", (field, from, to, label, readFrom, readTo) => {
    expect(read(change(field, from, to))).toMatchObject({ field: label, from: readFrom, to: readTo });
  });
});
