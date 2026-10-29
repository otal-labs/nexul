import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnersHealth } from "@/components/runner/RunnersHealth";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const runner = (id: string, connected: boolean, busy = false) => ({
  id, name: id, connected, last_seen: "2026-08-12T12:00:00Z", version: "", running_job: busy ? { id: `d-${id}`, kind: "deploy" } : null,
});

const serve = (runners: unknown[], queue: unknown[]) =>
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/runners" ? runners : queue }));

const renderHealth = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RunnersHealth />
    </QueryClientProvider>,
  );

describe("RunnersHealth", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("counts online, busy, offline and waiting in one line", async () => {
    serve([runner("a", true, true), runner("b", true), runner("c", false)], [{ id: "q-1", kind: "deploy" }]);
    renderHealth();

    expect(await screen.findByLabelText("Fleet health")).toHaveTextContent("2 of 3online1busy1offline1waiting");
  });

  it("leaves out a clause whose count is zero", async () => {
    serve([runner("a", true)], []);
    renderHealth();

    const line = await screen.findByLabelText("Fleet health");
    expect(line).toHaveTextContent("1 of 1online");
    expect(line).not.toHaveTextContent(/busy|offline|waiting/);
  });

  it("says there are no runners yet on an empty fleet", async () => {
    serve([], []);
    renderHealth();

    expect(await screen.findByLabelText("Fleet health")).toHaveTextContent("No runners yet");
  });
});
