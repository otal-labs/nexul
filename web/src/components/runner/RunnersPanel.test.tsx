import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnersPanel } from "@/components/runner/RunnersPanel";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: mocks.errorMessage }));

const renderPanel = (overrides?: { retry?: boolean }) =>
  render(
    <QueryClientProvider
      client={new QueryClient(overrides ? { defaultOptions: { queries: { retry: false } } } : undefined)}
    >
      <MemoryRouter>
        <RunnersPanel />
      </MemoryRouter>
    </QueryClientProvider>,
  );

const onlineRunner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: { id: "d-9", kind: "deploy", service: "api" },
  version: "v0.1.4",
  machine: "prod",
};

const idleRunner = {
  id: "r-2",
  name: "beta",
  connected: false,
  last_seen: "2026-08-12T11:00:00Z",
  running_job: null,
  version: "",
  machine: "prod",
};

const machine = {
  id: "m-1",
  name: "prod",
  stack_root: "/data/nexul",
  reported_hostname: "prod-host",
  first_seen: "2026-08-01T00:00:00Z",
  last_seen: "2026-08-12T12:00:00Z",
};

describe("RunnersPanel", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.errorMessage.mockReset();
  });

  it("groups runners under their machine, with fleet health and the queue", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/runners") return Promise.resolve({ data: [onlineRunner, idleRunner] });
      if (url === "/api/machines") return Promise.resolve({ data: [machine] });
      if (url === "/api/runners/queue") return Promise.resolve({ data: [{ id: "d-1", kind: "deploy", service: "web" }] });
      if (url === "/api/runners/latest-version") return Promise.resolve({ data: { version: "v0.1.4" } });
      return Promise.resolve({ data: [] });
    });
    renderPanel();

    expect(await screen.findByRole("heading", { name: "prod" })).toBeInTheDocument();
    expect(screen.getByText("prod-host")).toBeInTheDocument();
    expect(screen.getByText("2 runners")).toBeInTheDocument();
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("online")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByText("offline")).toBeInTheDocument();
    expect(screen.getByText("api")).toBeInTheDocument();
    expect(screen.getByText("d-1")).toBeInTheDocument();
    expect(screen.getByText("web")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /import from this machine/i })).toHaveAttribute(
      "href",
      "/wizard/project/import?machine=m-1",
    );
    expect(screen.getByRole("button", { name: /add a runner to this machine/i })).toBeInTheDocument();

    expect(screen.getByLabelText("Fleet health summary")).toBeInTheDocument();
    expect(screen.getByText("Online")).toBeInTheDocument();
    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(screen.getByText("#1")).toBeInTheDocument();
  });

  it("falls back to a flat runner list before any machine exists", async () => {
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/runners") return Promise.resolve({ data: [{ ...onlineRunner, machine: undefined }] });
      if (url === "/api/machines") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
    renderPanel();

    expect(await screen.findByText("alpha")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "prod" })).not.toBeInTheDocument();
  });

  it("shows empty states when nothing is connected or queued", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    renderPanel();

    expect(await screen.findByText("No runners connected yet.")).toBeInTheDocument();
    expect(screen.getByText("Nothing queued.")).toBeInTheDocument();
  });

  it("shows the shared loading display while runner data loads", async () => {
    mocks.get.mockImplementation(() => new Promise(() => {}));
    renderPanel({ retry: false });

    expect(await screen.findByRole("status")).toBeInTheDocument();
  });

  it("surfaces load errors with the shared error display", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("boom");
    renderPanel({ retry: false });

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Failed to load runners")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();
  });
});
