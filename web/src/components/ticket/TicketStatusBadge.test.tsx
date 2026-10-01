import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

const statuses = [
  { id: "st-backlog", name: "Backlog", kind: "backlog", icon: "", position: 0 },
  { id: "st-review", name: "In review", kind: "review", icon: "CircleDot", position: 0 },
];

const renderBadge = (status: string) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TicketStatusBadge ticket={{ project_id: "p-1", status }} />
    </QueryClientProvider>,
  );

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockResolvedValue({ data: statuses });
});

describe("TicketStatusBadge", () => {
  it("shows the project's status name for a status id", async () => {
    renderBadge("st-review");
    expect(await screen.findByText("In review")).toBeInTheDocument();
    expect(screen.queryByText("st-review")).not.toBeInTheDocument();
  });

  it("shows a readable name for a ticket still holding a legacy status", async () => {
    renderBadge("in_progress");
    expect(await screen.findByText("In progress")).toBeInTheDocument();
  });

  it("shows Unknown status instead of an id the project does not have", async () => {
    renderBadge("cb722f8f-0000-4000-8000-000000000000");
    expect(await screen.findByText("Unknown status")).toBeInTheDocument();
    expect(screen.queryByText(/cb722f8f/)).not.toBeInTheDocument();
  });
});
