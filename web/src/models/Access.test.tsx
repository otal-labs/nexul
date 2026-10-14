import { describe, expect, it } from "vitest";

import type { Area } from "@nexul/client-core/permissions";

import { canOpenWizard, canRunWizard } from "@/models/Access";

const holding = (...areas: Area[]) => (area: Area) => areas.includes(area);

describe("canRunWizard", () => {
  it.each([
    ["adding a service takes the scan, then the project, stack and deploy writes", false, ["newProject", "editProjects", "editStacks", "editDeploys"], true],
    ["adding a service without deploys is refused at the first deploy", false, ["newProject", "editProjects", "editStacks"], false],
    ["adding a service without projects:write is refused at the repository", false, ["newProject", "editStacks", "editDeploys"], false],
    ["adding a service without projects:create is refused at the scan", false, ["editProjects", "editStacks", "editDeploys"], false],
    ["a new project takes creating it and its setup, and no service writes", true, ["newProject", "editProjects"], true],
    ["a new project without creating it is refused at the first step", true, ["editProjects", "editStacks", "editDeploys"], false],
    ["a new project without its setup write is refused at Finish", true, ["newProject"], false],
  ] as const)("%s", (_, creates, areas, want) => {
    expect(canRunWizard(holding(...areas), creates)).toBe(want);
  });
});

describe("canOpenWizard", () => {
  it.each([
    ["a role that can only create projects", ["newProject", "editProjects"], true],
    ["a role that can only add services", ["newProject", "editProjects", "editStacks", "editDeploys"], true],
    ["a role that can do neither", ["editStacks", "editDeploys"], false],
  ] as const)("opens for %s", (_, areas, want) => {
    expect(canOpenWizard(holding(...areas))).toBe(want);
  });
});
