import { describe, expect, it } from "vitest";

import { lightsDot, showsDot, type WelcomeStepProgress } from "@nexul/client-core/welcome";

const step = (overrides: Partial<WelcomeStepProgress>): WelcomeStepProgress => ({ key: "github", state: "left", given: false, ...overrides });

describe("the dot rule", () => {
  it.each([
    ["a step left lights it", step({}), true],
    ["a done step doesn't", step({ state: "done" }), false],
    ["a skipped step doesn't", step({ state: "skipped" }), false],
    ["projects left but never given doesn't", step({ key: "projects" }), false],
    ["projects left once given does", step({ key: "projects", given: true }), true],
    ["projects given and skipped doesn't", step({ key: "projects", given: true, state: "skipped" }), false],
  ])("%s", (_, s, lit) => {
    expect(lightsDot(s)).toBe(lit);
  });

  it("shows while any step lights it", () => {
    expect(showsDot({ first_run_done: true, dot: false, steps: [step({ state: "done" }), step({ key: "projects" })] })).toBe(false);
    expect(showsDot({ first_run_done: true, dot: false, steps: [step({ state: "done" }), step({ key: "phone" })] })).toBe(true);
  });
});
