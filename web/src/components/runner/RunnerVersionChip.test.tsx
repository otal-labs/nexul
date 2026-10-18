import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnerVersionChip } from "@/components/runner/RunnerVersionChip";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const serverVersion = (version: string) => ({
  version,
  channel: "stable",
  latest: null,
  update_available: false,
});

const renderChip = (version: string) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RunnerVersionChip version={version} />
    </QueryClientProvider>,
  );

describe("RunnerVersionChip", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("shows the runner's version with an updating badge when it is behind the server version", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("v0.2.0") });
    renderChip("v0.1.4");
    expect(await screen.findByText("v0.1.4")).toBeInTheDocument();
    expect(await screen.findByText("updating · v0.2.0")).toBeInTheDocument();
  });

  it.each([
    ["the runner already matches the server version", "v0.2.0", "v0.2.0"],
    ["the runner is a local dev build", "dev", "v0.2.0"],
    ["the server itself is a dev build", "v0.1.4", "dev"],
  ])("shows no badge when %s", async (_name, runner, server) => {
    mocks.get.mockResolvedValue({ data: serverVersion(server) });
    renderChip(runner);
    await screen.findByText(runner);
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });

  it("shows no badge when the server version lookup errors", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    renderChip("v0.1.4");
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/version"));
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });
});
