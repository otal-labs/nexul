import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocTicketsRunLine } from "@/components/doc/DocTicketsRunLine";
import type { Ticket } from "@/models/Ticket";
import type { Trail } from "@/models/Trail";

const mocks = vi.hoisted(() => ({ trails: vi.fn(), tickets: vi.fn() }));

const RUN_START = "2026-10-20T10:00:00Z";
const run = (state: Trail["state"]) => ({ id: "t-1", play_id: "pl-t", state, started_at: RUN_START }) as Trail;
const ticket = (id: string, created_at: string) => ({ id, doc_id: "d-1", created_at }) as Ticket;

vi.mock("@/hooks/TrailHooks", () => ({ useFetchTrails: () => ({ data: mocks.trails() }) }));
vi.mock("@/hooks/TicketHooks", () => ({ useFetchTicketsByDoc: () => ({ data: mocks.tickets() }) }));
vi.mock("@/hooks/PlayHooks", () => ({
  useFetchApplicablePlays: () => ({ data: [{ id: "pl-t", builtin_key: "to-tickets-via-ai" }] }),
}));
vi.mock("@/hooks/ProjectHooks", () => ({ useFetchProject: () => ({ data: { id: "p-1", prefix: "WEB" } }) }));

const renderLine = () =>
  render(
    <MemoryRouter>
      <DocTicketsRunLine workspaceId="ws-1" docId="d-1" projectId="p-1" />
    </MemoryRouter>,
  );

beforeEach(() => {
  mocks.trails.mockReset();
  mocks.tickets.mockReset();
});

describe("DocTicketsRunLine", () => {
  it("says how many tickets the finished run wrote and links to the board", () => {
    mocks.trails.mockReturnValue([run("done")]);
    mocks.tickets.mockReturnValue([ticket("a", "2026-10-20T10:01:00Z"), ticket("b", "2026-10-20T10:02:00Z"), ticket("old", "2026-10-19T09:00:00Z")]);
    renderLine();
    expect(screen.getByText("2 tickets written")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "· Open board" })).toHaveAttribute("href", expect.stringMatching(/\/board\/WEB$/));
  });

  it("counts tickets as they land while the run is still writing", () => {
    mocks.trails.mockReturnValue([run("running")]);
    mocks.tickets.mockReturnValue([ticket("a", "2026-10-20T10:01:00Z")]);
    renderLine();
    expect(screen.getByText("Writing tickets")).toBeInTheDocument();
    expect(screen.getByText("· 1 so far")).toBeInTheDocument();
  });

  it("offers no board link when the run wrote nothing", () => {
    mocks.trails.mockReturnValue([run("failed")]);
    mocks.tickets.mockReturnValue([ticket("old", "2026-10-19T09:00:00Z")]);
    renderLine();
    expect(screen.getByText("Writing tickets failed")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "· Open board" })).not.toBeInTheDocument();
  });

  it("renders nothing before the doc has been turned into tickets", () => {
    mocks.trails.mockReturnValue([{ ...run("done"), play_id: "pl-other" }]);
    mocks.tickets.mockReturnValue([]);
    const { container } = renderLine();
    expect(container).toBeEmptyDOMElement();
  });
});
