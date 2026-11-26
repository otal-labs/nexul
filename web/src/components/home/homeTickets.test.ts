import { describe, expect, it } from "vitest";

import { ticketsOnYou } from "@/components/home/homeTickets";
import { StatusKind, type BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const status = (id: string, kind: StatusKind): BoardStatus => ({ id, name: id, position: 0, kind, icon: "", created_at: "", updated_at: "" });
const statuses = [status("todo", StatusKind.Backlog), status("doing", StatusKind.Progress), status("qa", StatusKind.Testing), status("shipped", StatusKind.Done)];

const ticket = (id: string, overrides: Partial<Ticket> = {}): Ticket => ({
  id, project_id: "p-1", category_id: "", type_id: "", title: id, body: "", status: "doing", position: 0, number: 1, doc_id: "",
  developer: "", developer_id: "", tester: "", tester_id: "", reporter: { kind: "user" } as Ticket["reporter"], due_on: "",
  created_at: "", updated_at: "2026-11-01T00:00:00Z", labels: null, ...overrides,
});

const me = { id: "u-sam", login: "sam" };

describe("ticketsOnYou", () => {
  it("keeps the tickets whose next move is the account's and drops done ones", () => {
    const list = [
      ticket("mine", { developer_id: "u-sam", developer: "sam" }),
      ticket("theirs", { developer_id: "u-lena", developer: "lena" }),
      ticket("done", { developer_id: "u-sam", status: "shipped" }),
      ticket("testing-mine", { status: "qa", developer_id: "u-lena", tester_id: "u-sam" }),
      ticket("testing-dev", { status: "qa", developer_id: "u-sam", tester_id: "u-lena" }),
      ticket("by-login", { developer: "sam" }),
    ];

    expect(ticketsOnYou(list, statuses, me).map((t) => t.ticket.id).sort()).toEqual(["by-login", "mine", "testing-mine"]);
  });

  it("puts work in motion before the backlog, newest first within a stage", () => {
    const list = [
      ticket("backlog", { status: "todo", developer_id: "u-sam", updated_at: "2026-11-09T00:00:00Z" }),
      ticket("older", { developer_id: "u-sam", updated_at: "2026-11-02T00:00:00Z" }),
      ticket("testing", { status: "qa", tester_id: "u-sam", updated_at: "2026-11-08T00:00:00Z" }),
      ticket("newer", { developer_id: "u-sam", updated_at: "2026-11-05T00:00:00Z" }),
    ];

    expect(ticketsOnYou(list, statuses, me).map((t) => t.ticket.id)).toEqual(["newer", "older", "testing", "backlog"]);
  });
});
