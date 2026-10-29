// @vitest-environment node
import { describe, expect, it } from "vitest";

import { mentionChipText, type MentionChipData } from "@nexul/client-core/mention";

const chip: MentionChipData = {
  type: "ticket",
  id: "t-1",
  title: "Fix login redirect loop",
  status_label: "In progress",
  can_open: true,
  project_prefix: "ERF",
  project_number: 1,
  type_label: "bug",
  developer_label: "Sam",
  due_label: "Aug 30",
};

describe("mentionChipText", () => {
  it("separates fields that only whitespace divides with a middle dot", () => {
    const text = mentionChipText("ticket", "{ticket.Project} {ticket.Ticket} {ticket.Status} {ticket.Type} {ticket.Developer} {ticket.Due}", chip, "");
    expect(text).toBe("ERF-1 · Fix login redirect loop · In progress · bug · Sam · Aug 30");
  });

  it("keeps the text the format puts between fields", () => {
    expect(mentionChipText("ticket", "{ticket.Project}: {ticket.Ticket} ({ticket.Status})", chip, "")).toBe("ERF-1: Fix login redirect loop (In progress)");
  });

  it("drops an empty field with its dot, so no dot stands alone", () => {
    const unassigned = { ...chip, developer_label: "" };
    expect(mentionChipText("ticket", "{ticket.Ticket} {ticket.Developer} {ticket.Due}", unassigned, "")).toBe("Fix login redirect loop · Aug 30");
  });

  it("leaves a field it does not know as typed", () => {
    expect(mentionChipText("ticket", "{ticket.Ticket} {ticket.Owner}", chip, "")).toBe("Fix login redirect loop · {ticket.Owner}");
  });

  it("falls back to the default format and the stored label while unresolved", () => {
    expect(mentionChipText("ticket", "", undefined, "Old title")).toBe("Old title");
  });

  it("shows a doc's title whatever the format", () => {
    expect(mentionChipText("doc", "{ticket.Project}", { ...chip, type: "doc", title: "Runbook" }, "")).toBe("Runbook");
  });
});
