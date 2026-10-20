// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Project } from "@/models/Project";
import {
  railMark,
  railRows,
  setupResumePath,
  wizardCanSkip,
  wizardOrder,
  wizardSkipPlan,
  wizardStepView,
  WizardSteps,
  type WizardDoor,
  type WizardFacts,
  type WizardStepId,
} from "@/models/ProjectWizard";

const doors: WizardDoor[] = ["setup", "add", "attach"];
const scans: WizardFacts["scan"][] = ["idle", "running", "failed", "empty"];
const bools = [false, true];

const everyFacts = (): WizardFacts[] =>
  doors.flatMap((door) =>
    bools.flatMap((project) =>
      bools.flatMap((stack) =>
        bools.flatMap((appRepo) => bools.flatMap((candidate) => scans.map((scan) => ({ door, project, stack, appRepo, candidate, scan })))),
      ),
    ),
  );

// Follows "needs" until a step with a view of its own; undefined when it loops or runs past the door's steps.
const settle = (start: WizardStepId, facts: WizardFacts): WizardStepId | undefined => {
  let step = start;
  for (let hops = 0; hops <= wizardOrder(facts.door, true).length; hops++) {
    const view = wizardStepView(step, facts);
    if (view.kind !== "needs") return step;
    step = view.needs;
  }
  return undefined;
};

describe("wizardStepView", () => {
  it("never sends a step to itself, and every chain of needs ends on a step that shows something", () => {
    for (const facts of everyFacts()) {
      for (const step of WizardSteps) {
        const view = wizardStepView(step, facts);
        if (view.kind === "needs") expect(view.needs, `${step} ${JSON.stringify(facts)}`).not.toBe(step);
        expect(settle(step, facts), `${step} ${JSON.stringify(facts)}`).toBeDefined();
      }
    }
  });

  it("only names a step the door has", () => {
    for (const facts of everyFacts()) {
      for (const step of WizardSteps) {
        const view = wizardStepView(step, facts);
        if (view.kind === "needs") expect(wizardOrder(facts.door, true), `${step} ${JSON.stringify(facts)}`).toContain(view.needs);
      }
    }
  });

  it("lets a project with an attached repository and no service always deploy it: Deploy and Service scan, show the scan, or show the choice", () => {
    for (const facts of everyFacts().filter((f) => f.door === "setup" && f.project && f.appRepo && !f.stack)) {
      for (const step of ["deploy", "service"] as const) {
        expect(wizardStepView(step, facts).kind, `${step} ${JSON.stringify(facts)}`).not.toBe("needs");
      }
    }
  });

  it("shows the scan's own state while there is no candidate, and the form once there is one", () => {
    const base: WizardFacts = { door: "setup", project: true, stack: false, appRepo: true, candidate: false, scan: "running" };
    expect(wizardStepView("service", base)).toEqual({ kind: "scanning" });
    expect(wizardStepView("service", { ...base, scan: "failed" })).toEqual({ kind: "scanFailed" });
    expect(wizardStepView("deploy", { ...base, scan: "empty" })).toEqual({ kind: "nothingFound" });
    expect(wizardStepView("deploy", { ...base, candidate: true })).toEqual({ kind: "form" });
    expect(wizardStepView("deploy", { ...base, stack: true })).toEqual({ kind: "summary" });
  });
});

describe("skipping", () => {
  const order = wizardOrder("setup", true);

  it("in the setup door, skipping Service or any step after it with no service goes straight to Done, skipping the rest", () => {
    for (const step of ["service", "env", "reach", "branches"] as const) {
      const plan = wizardSkipPlan(step, "setup", order, false);
      expect(plan.to).toBe("done");
      expect([step, ...plan.alsoSkipped].sort()).toEqual(["branches", "env", "reach", "service"]);
    }
  });

  it("moves on one step when a service exists, or outside the setup door", () => {
    expect(wizardSkipPlan("reach", "setup", order, true)).toEqual({ to: "branches", alsoSkipped: [] });
    expect(wizardSkipPlan("env", "add", wizardOrder("add", true), false)).toEqual({ to: "reach", alsoSkipped: [] });
  });

  it("leaves Service without a way past it in the doors that exist to make one", () => {
    expect(wizardCanSkip("service", "setup")).toBe(true);
    expect(wizardCanSkip("service", "add")).toBe(false);
    expect(wizardCanSkip("service", "attach")).toBe(false);
  });
});

describe("the rail", () => {
  const setup = { finished: false, steps: { project: "done", people: "done", reach: "skipped" } } as Project["setup"];
  const context = (door: WizardDoor, step: WizardStepId) => ({ door, order: wizardOrder(door, false), step, setup, stack: false });

  it("shows the recorded marks in the setup door", () => {
    expect(railMark("project", context("setup", "repository"))).toBe("done");
    expect(railMark("reach", context("setup", "repository"))).toBe("skipped");
  });

  it("shows only what this run passed in the add and attach doors, never the project's recorded marks", () => {
    expect(railMark("repository", context("add", "env"))).toBe("left");
    expect(railMark("repository", context("add", "reach"))).toBe("done");
    expect(railMark("reach", context("add", "repository"))).toBe("left");
    expect(railMark("service", context("attach", "branches"))).toBe("done");
  });

  it("reads Deploy from the service: done once one exists, later when setup skipped it", () => {
    const withService = { ...context("setup", "done"), setup: { finished: false, steps: { service: "skipped" } } as Project["setup"] };
    expect(railMark("deploy", withService)).toBe("later");
    expect(railMark("deploy", { ...withService, stack: true })).toBe("done");
  });

  it("nests the steps after Deploy in the setup door, shown only while Deploy's group is open or a service exists", () => {
    const order = wizardOrder("setup", false);
    const ids = (step: WizardStepId, stack: boolean) => railRows("setup", order, step, stack).map((row) => row.id);
    expect(ids("repository", false)).toEqual(["project", "people", "repository", "deploy", "done"]);
    expect(ids("service", false)).toEqual(["project", "people", "repository", "deploy", "service", "reach", "branches", "done"]);
    expect(ids("done", true)).toContain("service");
    expect(railRows("setup", order, "service", false).filter((row) => row.nested).map((row) => row.id)).toEqual(["service", "reach", "branches"]);
    expect(railRows("add", wizardOrder("add", false), "repository", false).some((row) => row.nested)).toBe(false);
  });
});

describe("setupResumePath", () => {
  const project = (setup: Partial<Project["setup"]>) => ({ id: "p-1", setup: { finished: false, steps: {}, ...setup } }) as Project;

  it("opens Deploy where the first open step needs a service that was never made", () => {
    const steps = { project: "done", people: "done", repository: "done" } as const;
    expect(setupResumePath(project({ steps }))).toBe("/wizard/project/deploy?project=p-1");
    expect(setupResumePath(project({ steps: { ...steps, service: "done" }, stack_id: "s-1" }))).toBe("/wizard/project/reach?project=p-1");
  });

  it("opens the first step neither done nor skipped, and Done when none is left", () => {
    expect(setupResumePath(project({ steps: { project: "done" } }))).toBe("/wizard/project/people?project=p-1");
    const all = Object.fromEntries(["project", "people", "repository", "service", "reach", "branches"].map((id) => [id, "done"])) as Project["setup"]["steps"];
    expect(setupResumePath(project({ steps: all }))).toBe("/wizard/project/done?project=p-1");
  });
});
