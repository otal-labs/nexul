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

  it("shows the runner's version", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("v0.2.0") });
    renderChip("v0.1.4");
    expect(await screen.findByText("v0.1.4")).toBeInTheDocument();
  });

  it("shows an updating badge when the runner is behind the server version", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("v0.2.0") });
    renderChip("v0.1.4");
    expect(await screen.findByText("updating · v0.2.0")).toBeInTheDocument();
  });

  it("shows no badge when the runner already matches the server version", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("v0.2.0") });
    renderChip("v0.2.0");
    await screen.findByText("v0.2.0");
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });

  it("shows no badge for a local dev runner build", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("v0.2.0") });
    renderChip("dev");
    await screen.findByText("dev");
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });

  it("shows no badge when the server itself is a dev build", async () => {
    mocks.get.mockResolvedValue({ data: serverVersion("dev") });
    renderChip("v0.1.4");
    await screen.findByText("v0.1.4");
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });

  it("shows no badge when the server version lookup errors", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    renderChip("v0.1.4");
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/version"));
    expect(screen.queryByText(/updating/)).not.toBeInTheDocument();
  });
});
