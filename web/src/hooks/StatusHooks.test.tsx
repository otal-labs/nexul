import { describe, expect, it } from "vitest";

import { statusFollower } from "@/hooks/StatusHooks";
import type { BoardStatus } from "@/models/Status";
import type { TicketStatus } from "@/models/Ticket";
import type { TicketLinkSet } from "@/models/TicketLink";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const doing: BoardStatus = { id: "st-1", name: "Doing", position: 1, kind: "progress", icon: "", created_at: "", updated_at: "" };
const done: BoardStatus = { ...doing, id: "st-2", name: "Done", position: 2, kind: "done" };

const naming = (status: string): TicketLinkSet => ({
  found_in: null,
  origin_unknown: false,
  bugs_found: [],
  blocked_by: [{ id: "t-2", project_id: "p-1", prefix: "ACME", number: 2, title: "Login", status: status as TicketStatus, done: false }],
  blocks: [],
  blocked: true,
});

const board = () =>
  seeded([
    [["getProjectStatuses", "p-1"], [doing, done]],
    [["getProjectStatuses", "p-2"], [doing]],
    [["getTicketLinkSet", "t-1"], naming("st-1")],
    [["getBlockers"], {}],
  ]);

describe("the status follower", () => {
  it("renames a column in place, and leaves the link views alone while its stage holds", async () => {
    const client = board();
    await followFrame(statusFollower, "status.updated", { status: { ...doing, project_id: "p-1", name: "Building" }, previous_kind: "progress" }, client);
    expect(client.getQueryData<BoardStatus[]>(["getProjectStatuses", "p-1"])?.map((s) => s.name)).toEqual(["Building", "Done"]);
    expect([["getProjectStatuses", "p-1"], ["getProjectStatuses", "p-2"], ["getTicketLinkSet", "t-1"], ["getBlockers"]].map((key) => isStale(client, key))).toEqual([
      false,
      false,
      false,
      false,
    ]);
  });

  it("refetches the column order and the link views naming its tickets when a column changes stage", async () => {
    const client = board();
    await followFrame(statusFollower, "status.updated", { status: { ...doing, project_id: "p-1", kind: "done" }, previous_kind: "progress" }, client);
    expect([["getProjectStatuses", "p-1"], ["getTicketLinkSet", "t-1"], ["getBlockers"]].map((key) => isStale(client, key))).toEqual([true, true, true]);
    expect(isStale(client, ["getProjectStatuses", "p-2"])).toBe(false);
  });

  it("leaves the link views alone for a renamed column whose board is not open, since the frame names its stage before", async () => {
    const client = seeded([
      [["getTicketLinkSet", "t-1"], naming("st-1")],
      [["getBlockers"], {}],
    ]);
    await followFrame(statusFollower, "status.updated", { status: { ...doing, project_id: "p-1", name: "Building" }, previous_kind: "progress" }, client);
    expect([isStale(client, ["getTicketLinkSet", "t-1"]), isStale(client, ["getBlockers"])]).toEqual([false, false]);
  });

  it("drops a deleted column from its project's board", async () => {
    const client = board();
    await followFrame(statusFollower, "status.deleted", { status: { ...done, project_id: "p-1" } }, client);
    expect(client.getQueryData<BoardStatus[]>(["getProjectStatuses", "p-1"])?.map((s) => s.id)).toEqual(["st-1"]);
  });
});
