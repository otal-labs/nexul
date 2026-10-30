import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationRunsFeed } from "@/components/automation/AutomationRunsFeed";
import { AutomationKind, AutomationRunOutcome } from "@/enums/Automation";
import type { Automation } from "@/models/Automation";
import type { AutomationRun } from "@/models/AutomationRun";
import { formatClockTime } from "@/utils/TimeUtility";

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

const automation: Automation = {
  id: "a1",
  workspace_id: "ws-1",
  name: "Ticket finished",
  description: "",
  kind: AutomationKind.Default,
  enabled: true,
  subscriptions: ["ticket.pr_merged"],
  config_schema: {},
  config_values: {},
  scopes: [],
  host_id: null,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
};

// The list endpoint answers with runs, or failures when asked for outcome=failed; one run's endpoint answers with that run.
const serve = (runs: AutomationRun[], failures: AutomationRun[] = []) =>
  mocks.get.mockImplementation(async (url: string, config?: { params?: { outcome?: string } }) => {
    if (url === "/api/automations/a1/runs") return { data: config?.params?.outcome === "failed" ? failures : runs };
    return { data: runs.find((r) => url.endsWith(`/${r.id}`)) };
  });

const renderFeed = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <AutomationRunsFeed automation={automation} />
    </QueryClientProvider>,
  );
};

describe("AutomationRunsFeed", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("says which event the first run waits for", async () => {
    serve([]);
    renderFeed();
    expect(await screen.findByText("No runs yet. It runs when ticket.pr_merged arrives.")).toBeInTheDocument();
  });

  it("lists a run under its day with its outcome, duration and clock time", async () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(new Date("2026-08-27T12:05:00Z"));
    serve([run()]);
    renderFeed();
    expect(await screen.findByText("ticket.pr_merged")).toBeInTheDocument();
    expect(screen.getByText("Today")).toBeInTheDocument();
    expect(screen.getByText("Success")).toBeInTheDocument();
    expect(screen.getByText(formatClockTime("2026-08-27T12:00:00Z"))).toBeInTheDocument();
    expect(screen.getByText("1 run · 0 failed")).toBeInTheDocument();
  });

  it("puts a failed run's error on its second line", async () => {
    serve([run({ outcome: AutomationRunOutcome.Failure, error: "403: tickets:write missing" })]);
    renderFeed();
    expect(await screen.findByText("403: tickets:write missing")).toBeInTheDocument();
    expect(screen.getByText("1 run · 1 failed")).toBeInTheDocument();
  });

  it("asks the server for failures only under Failures, and says when there are none", async () => {
    serve([run()]);
    const user = userEvent.setup();
    renderFeed();
    await screen.findByText("ticket.pr_merged");

    await user.click(screen.getByRole("radio", { name: "Failures" }));

    expect(await screen.findByText("No failures in the last 30 days.")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/automations/a1/runs", { params: { outcome: "failed" } });
  });

  it("opens the run detail sheet with logs on click", async () => {
    serve([run()]);
    const user = userEvent.setup();
    renderFeed();

    await user.click(await screen.findByText("ticket.pr_merged"));

    expect(mocks.get).toHaveBeenCalledWith("/api/automations/a1/runs/r1");
    expect(await screen.findByText(/handler ran/)).toBeInTheDocument();
  });
});
