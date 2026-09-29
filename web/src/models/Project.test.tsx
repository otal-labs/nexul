import { describe, expect, it } from "vitest";

import { projectTokenFromPath, SaveProjectFormSchema, switchProjectPath, type Project } from "@/models/Project";

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

describe("SaveProjectFormSchema prefix", () => {
  it.each(["P1", "PH", "PH1", "V2API", "nx"])("accepts %s", (prefix) => {
    expect(SaveProjectFormSchema.safeParse({ name: "Phase 1", prefix, icon: "" }).success).toBe(true);
  });

  it.each(["1P", "P", "P-1", "PHASE1X", ""])("rejects %j with the prefix message", (prefix) => {
    const result = SaveProjectFormSchema.safeParse({ name: "Phase 1", prefix, icon: "" });
    expect(result.error?.issues[0]?.message).toBe("Prefix is 2–5 letters or digits, starting with a letter");
  });
});
