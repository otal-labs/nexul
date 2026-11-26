// @vitest-environment node
import { describe, expect, it } from "vitest";

import { newDraft, subjectOf, toDraft, toRequest, toSaveRequest, type AutoPlay } from "@/models/AutoPlay";

describe("subjectOf", () => {
  it("reads a doc play and a stack play as themselves and every other play as a ticket play", () => {
    expect(["ticket", "doc", "stack", "interview"].map((type) => subjectOf({ type: type as never }))).toEqual(["ticket", "doc", "stack", "ticket"]);
  });
});

describe("a new auto play's run-on", () => {
  it("starts on what each play type can run on", () => {
    expect(newDraft("ticket").run_on).toBe("developer");
    expect(newDraft("doc").run_on).toBe("causer");
    expect(newDraft("stack").run_on).toBe("person");
    expect(newDraft("stack").moment).toBe("stack.service_crashed");
  });
});

describe("an auto play from a payload that still says computer", () => {
  const legacy = {
    run_on: "computer",
    run_as: "u-me",
    conditions: { match: "all", groups: [] },
    priority: { rules: [], otherwise: "normal" },
    memory_ids: [],
  } as unknown as AutoPlay;

  it("reads as a person through a draft and a switch, and sends no computers", () => {
    expect(toDraft(legacy, "stack").run_on).toBe("person");
    expect(toRequest(legacy)).toMatchObject({ run_on: "person" });
    expect(toRequest(legacy)).not.toHaveProperty("computer_ids");
  });

  it("sends run_as_me only when the draft asks for it", () => {
    expect(toSaveRequest({ ...toDraft(legacy, "stack"), run_as_me: false })).not.toHaveProperty("run_as_me");
    expect(toSaveRequest({ ...toDraft(legacy, "stack"), run_as_me: true })).toMatchObject({ run_as_me: true });
  });
});
