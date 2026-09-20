import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useRunnerInstall, useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const runner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: { id: "d-9", kind: "deploy", service: "api" },
  version: "v0.1.4",
};

const Harness = ({ useFn }: { useFn: () => { isPending: boolean; isSuccess: boolean; isError: boolean; data?: unknown } }) => {
  const query = useFn();
  return (
    <span data-testid="state">
      {query.isPending ? "loading" : query.isSuccess ? "loaded" : "error"}
    </span>
  );
};

describe("RunnerHooks", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("fetches runners", async () => {
    mocks.get.mockResolvedValue({ data: [runner] });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness useFn={useRunners} />
      </QueryClientProvider>,
    );
    expect(mocks.get).toHaveBeenCalledWith("/api/runners");
    await screen.findByText("loaded");
  });

  it("fetches the queue", async () => {
    mocks.get.mockResolvedValue({ data: [{ id: "d-1", kind: "deploy", service: "api" }] });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness useFn={useRunnerQueue} />
      </QueryClientProvider>,
    );
    expect(mocks.get).toHaveBeenCalledWith("/api/runners/queue");
    await screen.findByText("loaded");
  });

  it("surfaces errors without crashing", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <Harness useFn={useRunners} />
      </QueryClientProvider>,
    );
    await screen.findByText("error");
  });

  it("fetches the install payload when enabled", async () => {
    mocks.get.mockResolvedValue({
      data: {
        ws_url: "wss://deploy.example.com:8081/ws/runner",
        secret: "abc123",
        download_url: "https://deploy.example.com/api/runners/download",
      },
    });
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness useFn={() => useRunnerInstall(true)} />
      </QueryClientProvider>,
    );
    expect(mocks.get).toHaveBeenCalledWith("/api/runners/install");
    await screen.findByText("loaded");
  });

  it("does not fetch the install payload when disabled", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <Harness useFn={() => useRunnerInstall(false)} />
      </QueryClientProvider>,
    );
    expect(mocks.get).not.toHaveBeenCalled();
  });
});
