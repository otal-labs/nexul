import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { WelcomeProgress, WelcomeStepProgress } from "@nexul/client-core/welcome";

import { TeamGapLine } from "@/components/team/TeamGapLine";
import type { AccountStatus, TeamPerson } from "@/models/Team";

const step = (key: WelcomeStepProgress["key"], state: WelcomeStepProgress["state"]): WelcomeStepProgress => ({ key, state, given: false });

const person = (welcome: Partial<WelcomeProgress>, status: AccountStatus = "active"): TeamPerson => ({
  id: "u-bob", login: "bob", name: "Bob", avatar_url: "", status, created_at: "", providers: [], online: false, last_seen_at: null, workspaces: [],
  welcome: { first_run_done: true, steps: [], dot: false, in_a_project: true, ...welcome },
});

const lineOf = (p: TeamPerson) => render(<ul><li><TeamGapLine person={p} /></li></ul>).container.textContent;

describe("TeamGapLine", () => {
  it.each([
    ["names the one step left", person({ steps: [step("github", "done"), step("computer", "left")] }), "2 of 3 set up· computer not added"],
    ["names the first of several and counts the rest", person({ steps: [step("github", "left"), step("computer", "left"), step("phone", "left")] }), "1 of 4 set up· GitHub not connected +2"],
    ["says only that the wizard was never opened", person({ first_run_done: false, steps: [step("github", "left")] }), "0 of 2 set up· hasn't opened the wizard yet"],
    ["counts and names what was skipped, with nothing left", person({ steps: [step("github", "done"), step("phone", "skipped")] }), "2 of 3 set up· skipped phone app"],
    ["names a person in no project when no role gives the step", person({ in_a_project: false, steps: [step("github", "done")] }), "2 of 3 set up· not in a project yet"],
    ["each wording of a step left", person({ steps: [step("deploy_server", "left")] }), "1 of 2 set up· deploy server not added"],
    ["and the phone's", person({ steps: [step("phone", "left")] }), "1 of 2 set up· phone app not set up"],
  ])("%s", (_, p, want) => {
    expect(lineOf(p)).toBe(want);
  });

  it.each([
    ["every step is done", person({ steps: [step("github", "done"), step("computer", "done")] })],
    ["the account is disabled", person({ steps: [step("computer", "left")] }, "disabled")],
    ["the account is removed", person({ steps: [step("computer", "left")] }, "removed")],
    ["the server sent no progress", (() => {
      const p = person({});
      delete p.welcome;
      return p;
    })()],
  ])("shows nothing when %s", (_, p) => {
    expect(lineOf(p)).toBe("");
  });
});
