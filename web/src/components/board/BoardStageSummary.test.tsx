import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { BoardStageSummary } from "@/components/board/BoardStageSummary";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));

const status = (id: string, kind: BoardStatus["kind"]): BoardStatus =>
  ({ id, name: id, position: 0, kind, icon: "", created_at: "", updated_at: "" });

const ticket = (id: string, projectId: string, statusId: string) =>
  ({ id, project_id: projectId, status: statusId }) as Ticket;

describe("BoardStageSummary", () => {
  it("counts this project's tickets per stage and leaves out stages the project has no column for", async () => {
    mocks.get.mockImplementation((url: string) =>
      Promise.resolve({
        data: url.includes("statuses")
          ? [status("todo", "backlog"), status("doing", "progress"), status("wip", "progress"), status("shipped", "done")]
          : [
              ticket("1", "p-1", "todo"),
              ticket("2", "p-1", "doing"),
              ticket("3", "p-1", "wip"),
              ticket("4", "p-1", "shipped"),
              ticket("5", "p-2", "doing"),
            ],
      }),
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <BoardStageSummary projectId="p-1" />
      </QueryClientProvider>,
    );

    const summary = await screen.findByRole("group", { name: "Tickets by stage" });
    const stages = within(summary).getAllByRole("listitem").map((li) => li.textContent);
    expect(stages).toEqual(["Backlog1", "Progress2", "Done1"]);
  });
});
