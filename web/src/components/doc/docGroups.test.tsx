import { describe, expect, it } from "vitest";

import { groupDocsByDay } from "@/components/doc/docGroups";
import type { DocListItem } from "@/models/Doc";

const NOW = new Date(2026, 9, 1, 12, 0, 0);
const at = (daysAgo: number, hour = 9) => new Date(2026, 9, 1 - daysAgo, hour).toISOString();

const doc = (id: string, created: string, updated: string): DocListItem => ({
  id,
  project_id: "project-1",
  title: id,
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: created,
  updated_at: updated,
});

// EP01 was made first and edited today; EP06 and EP07 were made on later days and left alone.
const docs = [doc("EP06", at(3), at(3)), doc("EP01", at(5), at(0)), doc("EP07", at(2), at(2))];
const ids = (groups: ReturnType<typeof groupDocsByDay>) => groups.flatMap((g) => g.docs.map((d) => d.id));

describe("groupDocsByDay", () => {
  it("keeps an edited old doc in its created position when sorting by created", () => {
    expect(ids(groupDocsByDay(docs, "created_at", NOW))).toEqual(["EP07", "EP06", "EP01"]);
  });

  it("buckets by creation day when sorting by created", () => {
    const groups = groupDocsByDay(docs, "created_at", NOW);
    expect(groups.map((g) => g.label)).toEqual(["Earlier"]);
  });

  it("lifts an edited doc to the top and into Today when sorting by last edited", () => {
    const groups = groupDocsByDay(docs, "updated_at", NOW);
    expect(groups.map((g) => [g.label, g.docs.map((d) => d.id)])).toEqual([
      ["Today", ["EP01"]],
      ["Earlier", ["EP07", "EP06"]],
    ]);
  });

  it("splits Today, Yesterday and Earlier on the chosen timestamp", () => {
    const mixed = [doc("old", at(4), at(4)), doc("yday", at(1), at(1)), doc("new", at(0), at(0))];
    const groups = groupDocsByDay(mixed, "created_at", NOW);
    expect(groups.map((g) => [g.label, g.docs.map((d) => d.id)])).toEqual([
      ["Today", ["new"]],
      ["Yesterday", ["yday"]],
      ["Earlier", ["old"]],
    ]);
  });
});
