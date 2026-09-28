import { describe, expect, it } from "vitest";

import { projectTokenFromPath, switchProjectPath, type Project } from "@/models/Project";

describe("projectTokenFromPath", () => {
  it.each([
    ["/board/NEXUL", "NEXUL"],
    ["/projects/NEXUL/settings", "NEXUL"],
    ["/projects/NEXUL/interview", "NEXUL"],
    ["/docs/NEXUL/d-1", "NEXUL"],
    ["/memories/NEXUL/m-1", "NEXUL"],
    ["/board", undefined],
    ["/docs", undefined],
    ["/docs/d-1", undefined],
    ["/memories/m-1", undefined],
    ["/runners", undefined],
  ])("%s → %s", (pathname, want) => {
    expect(projectTokenFromPath(pathname)).toBe(want);
  });
});

describe("switchProjectPath", () => {
  const project = { id: "p-2", prefix: "FE" } as Project;

  it.each([
    ["/projects/BE/settings", "/projects/FE/settings"],
    ["/projects/BE/interview", "/projects/FE/interview"],
    ["/board/BE", "/board/FE"],
    ["/docs/BE/d-1", "/board/FE"],
    ["/runners", "/board/FE"],
  ])("%s → %s", (pathname, want) => {
    expect(switchProjectPath(pathname, project)).toBe(want);
  });
});
