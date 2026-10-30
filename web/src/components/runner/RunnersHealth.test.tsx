import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnersHealth } from "@/components/runner/RunnersHealth";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const runner = (id: string, connected: boolean, busy = false) => ({
  id, name: id, connected, last_seen: "2026-08-12T12:00:00Z", version: "", running_job: busy ? { id: `d-${id}`, kind: "deploy" } : null,
});

const serve = (runners: unknown[], queue: unknown[], codes: unknown[] = []) => {
  const data: Record<string, unknown[]> = { "/api/runners": runners, "/api/runners/queue": queue, "/api/runners/enrollments": codes };
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: data[url] }));
};

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

  it("counts online, busy, offline, unused codes and waiting in one line", async () => {
    serve([runner("a", true, true), runner("b", true), runner("c", false)], [{ id: "q-1", kind: "deploy" }], [{ id: "c-1", name: "next" }, { id: "c-2", name: "laptop", computer_id: "pc-1" }]);
    renderHealth();

    expect(await screen.findByLabelText("Fleet health")).toHaveTextContent("2 of 3 online1 busy1 offline1 unused code1 waiting");
  });

  it("leaves out a clause whose count is zero", async () => {
    serve([runner("a", true)], []);
    renderHealth();

    const line = await screen.findByLabelText("Fleet health");
    expect(line).toHaveTextContent("1 of 1 online");
    expect(line).not.toHaveTextContent(/busy|offline|code|waiting/);
  });

  it("says there are no runners yet on an empty fleet", async () => {
    serve([], []);
    renderHealth();

    expect(await screen.findByLabelText("Fleet health")).toHaveTextContent("No runners yet");
  });
});
