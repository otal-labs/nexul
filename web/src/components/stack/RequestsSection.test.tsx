import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RequestsSection } from "@/components/stack/RequestsSection";
import type { Stack, StackRequests } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: () => "boom" }));

const FROM = "2026-10-16T09:43:00.000Z";

const hour = (requests: (number | null)[], errors?: (number | null)[]): StackRequests => ({
  counted: true,
  from: FROM,
  requests,
  errors_5xx: errors ?? requests.map((r) => (r === null ? null : 0)),
});

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "box",
  strategy: "compose",
  managed: true,
  error_rate: { enabled: true, percent: 5, minutes: 5, min_requests: 20 },
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const renderSection = (value: Stack = stack) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <RequestsSection stack={value} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("RequestsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the hour's total, the 5xx share, and a bar for every minute", async () => {
    const requests = Array.from({ length: 60 }, (_, i) => (i % 2 === 0 ? 10 : 20));
    const errors = Array.from({ length: 60 }, (_, i) => (i === 59 ? 3 : 0));
    mocks.get.mockResolvedValue({ data: hour(requests, errors) });
    renderSection();

    expect(await screen.findByRole("heading", { name: "Requests" })).toBeInTheDocument();
    expect(screen.getByText("900")).toBeInTheDocument();
    expect(screen.getByText("5xx 0.3%")).toBeInTheDocument();
    expect(screen.getByText("(3)")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Requests per minute over the last hour" })).toBeInTheDocument();
    expect(screen.queryByText(/No request data/)).not.toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/stacks/stack-1/requests");
  });

  it("names when the data stopped once five minutes are missing, and keeps the earlier bars", async () => {
    const requests = [...Array.from({ length: 55 }, () => 2), ...Array.from({ length: 5 }, () => null)];
    mocks.get.mockResolvedValue({ data: hour(requests) });
    renderSection();

    expect(await screen.findByText(/^No request data since \d\d:\d\d$/)).toBeInTheDocument();
    expect(screen.getByText("110")).toBeInTheDocument();
  });

  it("does not call a short gap a stop", async () => {
    const requests = [...Array.from({ length: 56 }, () => 2), ...Array.from({ length: 4 }, () => null)];
    mocks.get.mockResolvedValue({ data: hour(requests) });
    renderSection();
    expect(await screen.findByText("112")).toBeInTheDocument();
    expect(screen.queryByText(/No request data/)).not.toBeInTheDocument();
  });

  it("has an honest empty state for a stack with a hostname and no counts yet", async () => {
    mocks.get.mockResolvedValue({ data: hour(Array.from({ length: 60 }, () => null)) });
    renderSection();

    expect(await screen.findByText(/^No request data yet/)).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("says there is no 5xx when there were none, and states the error-rate rule in the footer", async () => {
    mocks.get.mockResolvedValue({ data: hour(Array.from({ length: 60 }, () => 4)) });
    renderSection({ ...stack, error_rate: { enabled: true, percent: 2.5, minutes: 10, min_requests: 1000 } });
    expect(await screen.findByText("Alert when 5xx above 2.5% for 10 minutes, with at least 1,000 requests.")).toBeInTheDocument();
    expect(screen.getByText("No 5xx")).toBeInTheDocument();
  });

  it("says once that nothing is counted while the rule is off, linking to where to turn it on", async () => {
    mocks.get.mockResolvedValue({ data: hour(Array.from({ length: 60 }, () => null)) });
    renderSection({ ...stack, error_rate: { enabled: false, percent: 5, minutes: 5, min_requests: 20 } });

    expect(await screen.findByText(/^Not counted\. Turn on the error-rate rule in/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Alerts" })).toHaveAttribute("href", expect.stringMatching(/\/stacks\/stack-1\/settings\/alerts$/));
    expect(screen.queryByText(/^No error-rate rule on this stack/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^No request data yet/)).not.toBeInTheDocument();
  });

  it("draws nothing for a stack with no hostname", async () => {
    mocks.get.mockResolvedValue({ data: { counted: false, from: "0001-01-01T00:00:00Z", requests: [], errors_5xx: [] } });
    const { container } = renderSection();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    await vi.waitFor(() => expect(container).toBeEmptyDOMElement());
  });

  it("says the read failed instead of vanishing", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    renderSection();
    expect(await screen.findByText("Couldn't load requests.")).toBeInTheDocument();
  });
});
