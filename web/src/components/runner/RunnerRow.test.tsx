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
    expect(screen.getByText("online")).toBeInTheDocument();
    expect(screen.getByText(/last seen/i)).toBeInTheDocument();
    expect(screen.getByText("api")).toBeInTheDocument();
  });

  it("shows idle for a disconnected runner without a job", () => {
    renderRow({ runner: idleRunner });
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByText("offline")).toBeInTheDocument();
    expect(screen.getByText("idle")).toBeInTheDocument();
  });

  it("recedes offline runners at reduced opacity", () => {
    const { container } = renderRow({ runner: idleRunner });
    expect(container.querySelector("li")).toHaveClass("opacity-60");
    const { container: online } = renderRow({ runner: onlineRunner });
    expect(online.querySelector("li")).not.toHaveClass("opacity-60");
  });

  it("staggers the mount entrance by row index, capped at the eighth row", () => {
    const { container: third } = renderRow({ runner: onlineRunner, index: 2 });
    expect(third.querySelector("li > div")).toHaveStyle({ animationDelay: "48ms" });

    const { container: farDown } = renderRow({ runner: onlineRunner, index: 20 });
    expect(farDown.querySelector("li > div")).toHaveStyle({ animationDelay: "0ms" });
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
