import { describe, expect, it } from "vitest";

import { sourceName, sourcesChangedSince, sourcesMeta, type InterviewSource } from "@/models/InterviewSource";
import type { Trail } from "@/models/Trail";

const source = (patch: Partial<InterviewSource>): InterviewSource => ({
  id: "s-1", workspace_id: "ws-1", project_id: "p-1", kind: "path", ref: "practices", label: "practices", stance: "follow",
  added_by: "u-1", added_at: "2026-10-01T10:00:00Z", updated_at: "2026-10-01T10:00:00Z", ...patch,
});

const trail = (state: Trail["state"], started_at: string) => ({ state, started_at }) as Trail;

describe("sourcesChangedSince", () => {
  const runs = [trail("failed", "2026-10-03T10:00:00Z"), trail("done", "2026-10-02T10:00:00Z")];

  it("flags a source added, or a doc, memory, or text changed, after the last finished run started", () => {
    expect(sourcesChangedSince([source({ added_at: "2026-10-02T11:00:00Z" })], runs)).toBe(true);
    expect(sourcesChangedSince([source({ kind: "doc", ref_updated_at: "2026-10-02T11:00:00Z" })], runs)).toBe(true);
    expect(sourcesChangedSince([source({ kind: "text", updated_at: "2026-10-02T11:00:00Z" })], runs)).toBe(true);
  });

  it("can't tell for a path or a project, and says nothing before any run finished", () => {
    expect(sourcesChangedSince([source({ updated_at: "2026-10-02T11:00:00Z" })], runs)).toBe(false);
    expect(sourcesChangedSince([source({ added_at: "2026-10-02T11:00:00Z" })], [trail("failed", "2026-10-01T09:00:00Z")])).toBe(false);
  });
});

describe("naming", () => {
  it("names a gone or hidden source by its kind and counts stances", () => {
    expect(sourceName(source({ kind: "memory", gone: true, label: "" }))).toBe("Memory");
    expect(sourceName(source({ kind: "doc", not_visible: true, label: "" }))).toBe("Doc");
    expect(sourcesMeta([source({}), source({ stance: "question" })])).toBe("2 sources · 1 follow · 1 question");
    expect(sourcesMeta([])).toBe("None yet");
  });
});
