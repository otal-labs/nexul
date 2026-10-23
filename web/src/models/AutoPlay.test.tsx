// @vitest-environment node
import { describe, expect, it } from "vitest";

import { newDraft, subjectOf, toDraft, toRequest, type AutoPlay } from "@/models/AutoPlay";

describe("subjectOf", () => {
  it("reads a doc play and a stack play as themselves and every other play as a ticket play", () => {
    expect(["ticket", "doc", "stack", "interview"].map((type) => subjectOf({ type: type as never }))).toEqual(["ticket", "doc", "stack", "ticket"]);
  });
});

describe("a new auto play's run-on", () => {
  it("starts on what each play type can run on", () => {
    expect(newDraft("ticket").run_on).toBe("developer");
    expect(newDraft("doc").run_on).toBe("causer");
    expect(newDraft("stack").run_on).toBe("computer");
    expect(newDraft("stack").moment).toBe("stack.service_crashed");
  });
});

describe("an auto play from a payload that only names its first computer", () => {
  const legacy = {
    run_on: "computer",
    run_as: "u-me",
    computer_id: "c-1",
    conditions: { match: "all", groups: [] },
    priority: { rules: [], otherwise: "normal" },
    memory_ids: [],
  } as unknown as AutoPlay;

  it("keeps that computer as its list through a draft and a switch", () => {
    expect(toDraft(legacy, "stack").computer_ids).toEqual(["c-1"]);
    expect(toRequest(legacy).computer_ids).toEqual(["c-1"]);
  });
});
