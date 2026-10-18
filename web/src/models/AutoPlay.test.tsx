// @vitest-environment node
import { describe, expect, it } from "vitest";

import { newDraft, subjectOf, toSaveRequest } from "@/models/AutoPlay";

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

  it("sends the computer only while it runs on one", () => {
    const draft = { ...newDraft("ticket"), computer_id: "c-1" };
    expect(toSaveRequest(draft).computer_id).toBe("");
    expect(toSaveRequest({ ...draft, run_on: "computer" }).computer_id).toBe("c-1");
  });
});
