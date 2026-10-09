import { describe, expect, it } from "vitest";

import { templateFollower } from "@/hooks/TemplateHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the template follower", () => {
  it("refetches every view that shows a template's text when an instance template changes", async () => {
    const client = seeded([
      [["getTemplates"], []],
      [["getWorkspaces"], []],
      [["getInterviewTemplate", "ws-1"], {}],
      [["getWorkspacePlays", "ws-1"], []],
      [["getProjectTicketTypes", "p-1"], []],
      [["getProjectStatuses", "p-1"], []],
    ]);
    await followFrame(templateFollower, "instance_template.updated", { kind: "interview", key: "" }, client);
    const keys = [["getTemplates"], ["getWorkspaces"], ["getInterviewTemplate", "ws-1"], ["getWorkspacePlays", "ws-1"], ["getProjectTicketTypes", "p-1"], ["getProjectStatuses", "p-1"]];
    expect(keys.map((key) => isStale(client, key))).toEqual([true, true, true, true, true, false]);
  });
});
