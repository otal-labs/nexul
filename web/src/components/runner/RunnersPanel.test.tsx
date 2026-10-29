import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RunnersPanel } from "@/components/runner/RunnersPanel";
import { resetViewer, viewer } from "@/test/viewerActions";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

beforeEach(() => {
  resetViewer();
  viewer.workspace = ["stacks:write"];
});

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, delete: mocks.delete }, errorMessage: mocks.errorMessage }));

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

const busyRunner = {
  id: "r-1",
  name: "alpha",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  running_job: { id: "d-9", kind: "deploy", service: "api" },
  version: "v0.1.4",
  machine: "prod",
};

const offlineRunner = {
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
  actions: ["machines:write"],
};

const shapes: Record<string, unknown> = {
  "/api/team": { people: [], workspaces: [], can_manage_accounts: true },
  "/api/auth/me": { user: { id: "u-1" }, instance_permissions: [], instance_owner: true },
};

const serve = (data: Record<string, unknown>) =>
  mocks.get.mockImplementation((url: string) => {
    if (url in data) return Promise.resolve({ data: data[url] });
    if (url.startsWith("/api/permissions?")) return Promise.resolve({ data: { grants: [] } });
    return Promise.resolve({ data: shapes[url] ?? [] });
  });

describe("RunnersPanel", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.delete.mockReset().mockResolvedValue({});
    mocks.errorMessage.mockReset();
  });

  it("lists each machine's runners in one list under a column head, saying what each is doing", async () => {
    serve({ "/api/runners": [busyRunner, offlineRunner], "/api/machines": [machine], "/api/runners/queue": [] });
    renderPanel();

    const group = within(await screen.findByRole("region", { name: "prod" }));
    expect(group.getByText("prod-host")).toBeInTheDocument();
    expect(group.getByText("1 of 2 online")).toBeInTheDocument();
    expect(group.getByText("api")).toBeInTheDocument();
    expect(group.getByText(/^offline ·/)).toBeInTheDocument();
    expect(group.getByTitle("alpha")).toBeInTheDocument();
    expect(group.getByTitle("beta")).toBeInTheDocument();
    expect(screen.getByText("Runner").parentElement).toHaveTextContent("RunnerVersionDoing");
    expect(screen.queryByText(/^d-9$/)).not.toBeInTheDocument();
  });

  it("says why each queued job waits, and shows no waiting list when nothing waits", async () => {
    serve({
      "/api/runners": [busyRunner, { ...offlineRunner, machine: "lab" }],
      "/api/machines": [machine, { ...machine, id: "m-2", name: "lab" }],
      "/api/runners/queue": [
        { id: "d-1", kind: "deploy", service: "web", target: "prod" },
        { id: "d-2", kind: "build", target: "lab" },
      ],
    });
    renderPanel();

    const waiting = within(await screen.findByRole("region", { name: /Waiting for a runner/ }));
    expect(waiting.getByLabelText("Queue position 1").parentElement).toHaveTextContent("all runners busy");
    expect(waiting.getByLabelText("Queue position 2").parentElement).toHaveTextContent("no runner online");
    expect(waiting.getAllByText("build").length).toBeGreaterThan(0);
  });

  it("puts an unused code under its machine and revokes it from its menu", async () => {
    const user = userEvent.setup();
    serve({
      "/api/runners": [busyRunner],
      "/api/machines": [machine],
      "/api/runners/enrollments": [
        { id: "c-1", name: "build-box", machine: "prod", created_at: "2026-08-12T12:00:00Z", expires_at: "2026-08-13T12:00:00Z" },
        { id: "c-2", name: "computer-ab12cd34", computer_id: "pc-1", created_at: "2026-08-12T12:00:00Z", expires_at: "2026-08-13T12:00:00Z" },
      ],
    });
    renderPanel();

    const group = within(await screen.findByRole("region", { name: "prod" }));
    expect(await group.findByTitle("build-box")).toBeInTheDocument();
    expect(screen.queryByTitle("computer-ab12cd34")).not.toBeInTheDocument();

    await user.click(group.getByRole("button", { name: "More actions for the code for build-box" }));
    await user.click(await screen.findByRole("menuitem", { name: "Revoke code" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/runners/enrollments/c-1");
  });

  it("gathers runners and codes no machine holds in a last No machine group", async () => {
    serve({
      "/api/runners": [{ ...busyRunner, machine: undefined }],
      "/api/machines": [],
      "/api/runners/enrollments": [{ id: "c-3", name: "laptop", created_at: "2026-08-12T12:00:00Z", expires_at: "2026-08-13T12:00:00Z" }],
    });
    renderPanel();

    const group = within(await screen.findByRole("region", { name: "No machine" }));
    expect(group.getByTitle("alpha")).toBeInTheDocument();
    expect(await group.findByTitle("laptop")).toBeInTheDocument();
  });

  it("offers Add runner on a machine with no runners", async () => {
    serve({ "/api/runners": [], "/api/machines": [machine] });
    renderPanel();

    const group = within(await screen.findByRole("region", { name: "prod" }));
    expect(group.getByText("No runners yet")).toBeInTheDocument();
    expect(group.getAllByRole("button", { name: /add a runner to this machine/i })).toHaveLength(2);
  });

  it("shows the empty state when no machine or runner is visible", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    renderPanel();

    expect(await screen.findByText("No runners on the machines you can see yet.")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: /Waiting for a runner/ })).not.toBeInTheDocument();
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
    expect(screen.getByText("Couldn't load runners.")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();
  });
});
