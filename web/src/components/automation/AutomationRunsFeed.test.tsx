import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationRunsFeed } from "@/components/automation/AutomationRunsFeed";
import { AutomationRunOutcome } from "@/enums/Automation";
import type { AutomationRun } from "@/models/AutomationRun";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: () => "error",
}));

const run = (overrides: Partial<AutomationRun> = {}): AutomationRun => ({
  id: "r1",
  automation_id: "a1",
  event_topic: "ticket.pr_merged",
  event_id: "e1",
  outcome: AutomationRunOutcome.Success,
  started_at: "2026-08-27T12:00:00Z",
  finished_at: "2026-08-27T12:00:01Z",
  duration_ms: 340,
  logs: "handler ran\nreturned true",
  created_at: "2026-08-27T12:00:01Z",
  ...overrides,
});

const renderFeed = (runs: AutomationRun[]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationRunsFeed automationId="a1" runs={runs} />
    </QueryClientProvider>,
  );
};

describe("AutomationRunsFeed", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("shows an empty state with no runs", () => {
    renderFeed([]);
    expect(screen.getByText("No runs yet")).toBeInTheDocument();
  });

  it("lists runs with outcome, duration, and relative time", () => {
    renderFeed([run()]);
    expect(screen.getByText("ticket.pr_merged")).toBeInTheDocument();
    expect(screen.getByText("Success")).toBeInTheDocument();
    expect(screen.getByText("340ms")).toBeInTheDocument();
  });

  it("opens the run detail sheet with logs on click", async () => {
    mocks.get.mockResolvedValue({ data: run() });
    const user = userEvent.setup();
    renderFeed([run()]);

    await user.click(screen.getByText("ticket.pr_merged"));

    expect(mocks.get).toHaveBeenCalledWith("/api/automations/a1/runs/r1");
    expect(await screen.findByText(/handler ran/)).toBeInTheDocument();
  });
});
