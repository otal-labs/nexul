// @vitest-environment node
import { describe, expect, it } from "vitest";

import { overviewLists } from "@/components/doc/docsOverviewLists";
import type { DocListItem } from "@/models/Doc";

const doc = (id: string, day: number): DocListItem => ({
  id,
  project_id: "p-1",
  folder_id: "f-1",
  title: id,
  version: 1,
  archived: false,
  locked: false,
  can_open: true,
  created_at: "2026-01-01T09:00:00Z",
  updated_at: `2026-01-${String(day).padStart(2, "0")}T09:00:00Z`,
});

describe("overviewLists", () => {
  it("puts waiting docs before running ones, newest edit first in each, and keeps five", () => {
    const docs = [doc("run-old", 1), doc("wait-old", 2), doc("run-new", 9), doc("wait-new", 8), doc("idle", 10), doc("run-3", 3), doc("run-4", 4)];
    const states = ["running", "waiting", "running", "waiting", undefined, "running", "running"] as const;

    const { attention } = overviewLists(docs, [...states]);

    expect(attention.map((a) => a.doc.id)).toEqual(["wait-new", "wait-old", "run-new", "run-4", "run-3"]);
  });

  it("lists the twenty newest edits, whatever their state", () => {
    const docs = Array.from({ length: 25 }, (_, i) => doc(`d${i + 1}`, i + 1));

    const { recent } = overviewLists(docs, docs.map(() => undefined));

    expect(recent).toHaveLength(20);
    expect(recent[0]?.id).toBe("d25");
    expect(recent.at(-1)?.id).toBe("d6");
  });
});
