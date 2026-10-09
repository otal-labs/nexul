import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnerRow } from "@/components/runner/RunnerRow";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const onlineRunner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: { id: "d-9", kind: "deploy", service: "api" },
  version: "v0.1.4",
};

const idleRunner = {
  id: "r-2",
  name: "beta",
  connected: false,
  last_seen: "2026-08-12T11:00:00Z",
  running_job: null,
  version: "",
};

const renderRow = (props: Parameters<typeof RunnerRow>[0]) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RunnerRow {...props} />
    </QueryClientProvider>,
  );

describe("RunnerRow", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: { version: "v0.1.4" } });
  });

  it("shows name, status, and the running job for a connected runner", () => {
    renderRow({ runner: onlineRunner });
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByTitle("Last seen")).toBeInTheDocument();
    expect(screen.getByText("api")).toBeInTheDocument();
  });

  it("says offline, not idle, for a disconnected runner", () => {
    renderRow({ runner: idleRunner });
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByText("offline")).toBeInTheDocument();
    expect(screen.queryByText("idle")).not.toBeInTheDocument();
  });

  it("names a runner by its short id when it shares the machine's name", () => {
    renderRow({ runner: { ...idleRunner, id: "12aa2fd2-c2ac-7416", name: "prod" }, machineName: "prod" });
    expect(screen.getByText("runner 12aa2fd2")).toBeInTheDocument();
    expect(screen.queryByText("prod")).not.toBeInTheDocument();
  });

  it("shows the runner's build version when known", () => {
    renderRow({ runner: onlineRunner });
    expect(screen.getByText("v0.1.4")).toBeInTheDocument();
  });

  it("shows no version chip when the runner never reported one", () => {
    renderRow({ runner: idleRunner });
    expect(screen.queryByText("v0.1.4")).not.toBeInTheDocument();
  });
});
