import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { toast } from "sonner";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeployPage } from "@/pages/DeployPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: () => "boom",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const T0 = Date.parse("2026-09-22T10:00:00Z");

const stack = { id: "stack-1", project_id: "proj-1", name: "api", slug: "api", machine: "instance", strategy: "compose", managed: true };

const deploy = (overrides: Record<string, unknown>) => ({
  id: "d-1",
  kind: "build",
  stack_id: "stack-1",
  service: "api",
  target: "instance",
  image: "",
  status: "running",
  strategy: "compose",
  created_at: new Date(T0).toISOString(),
  updated_at: new Date(T0).toISOString(),
  ...overrides,
});

const lines = [
  { seq: 1, ts: T0 + 5_000, phase: "checkout", text: "Cloning into 'api'..." },
  { seq: 2, ts: T0 + 12_000, phase: "build", text: "$ bun install --frozen-lockfile" },
];

const mockApi = (d: Record<string, unknown>, log: unknown[]) =>
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/deploys/d-1") return Promise.resolve({ data: d });
    if (url === "/api/deploys/d-1/log") return Promise.resolve({ data: log });
    if (url === "/api/stacks/stack-1") return Promise.resolve({ data: stack });
    return Promise.resolve({ data: [] });
  });

const renderPage = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/stacks/stack-1/deploys/d-1"]}>
        <Routes>
          <Route path="/stacks/:stackId/deploys/:deployId" element={<DeployPage />} />
          <Route path="/stacks/:stackId" element={<p>stack page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("DeployPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date(T0 + 30_000));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the header, derived steps, and the log lines for a running build", async () => {
    mockApi(deploy({}), lines);
    renderPage();
    expect(await screen.findByRole("heading", { name: "api" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /back to deploy history/i })).toHaveAttribute("href", "/stacks/stack-1?section=history");
    expect(screen.getByText("d-1")).toBeInTheDocument();
    expect(screen.getByText("running")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Building and deploying" })).toBeInTheDocument();

    const steps = screen.getAllByRole("listitem").filter((li) => li.closest("ol") && !li.closest("[role='log']"));
    expect(steps.map((li) => li.textContent)).toEqual([
      "Waiting for a runnerdone5s",
      "Cloning repositorydone7s",
      "Buildingactive18s",
      "Deployingpending—",
    ]);

    const log = await screen.findByRole("log", { name: "Deploy log" });
    expect(within(log).getByText("Cloning into 'api'...")).toBeInTheDocument();
    expect(within(log).getByText("$ bun install --frozen-lockfile")).toBeInTheDocument();
    expect(within(log).getAllByText(/^\d{2}:\d{2}:\d{2}\.\d{3}$/)).toHaveLength(2);
    expect(log).toHaveAttribute("aria-live", "off");
    expect(screen.getByRole("status")).toHaveTextContent("Building and deploying: Building");
  });

  it("shows the cancel button only while the deploy is active", async () => {
    mockApi(deploy({}), lines);
    renderPage();
    expect(await screen.findByRole("button", { name: "Cancel deployment" })).toBeInTheDocument();
  });

  it("hides cancel, checks every started step, and titles a healthy deploy as deployed", async () => {
    mockApi(deploy({ status: "healthy", updated_at: new Date(T0 + 20_000).toISOString() }), lines);
    renderPage();
    expect(await screen.findByRole("heading", { name: "Deployed" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel deployment" })).not.toBeInTheDocument();
    expect(screen.getByText("Building").parentElement).toHaveTextContent("done8s");
    expect(screen.getByText("Deploying").parentElement).toHaveTextContent("—");
  });

  it("marks the last started step failed and titles the page accordingly", async () => {
    mockApi(deploy({ status: "failed", updated_at: new Date(T0 + 20_000).toISOString() }), [
      ...lines,
      { seq: 3, ts: T0 + 20_000, phase: "", text: "deploy failed: exit 1" },
    ]);
    renderPage();
    expect(await screen.findByRole("heading", { name: "Deploy failed" })).toBeInTheDocument();
    expect(screen.getByText("Building").parentElement).toHaveTextContent("failed");
    expect(within(await screen.findByRole("log")).getByText("deploy failed: exit 1")).toBeInTheDocument();
  });

  it("posts the cancel request when cancel is clicked", async () => {
    mockApi(deploy({}), lines);
    mocks.post.mockResolvedValue({ data: {} });
    renderPage();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    await user.click(await screen.findByRole("button", { name: "Cancel deployment" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/deploys/d-1/cancel");
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Cancel requested"));
  });

  it("copies the log as timestamped text and toasts", async () => {
    mockApi(deploy({}), lines);
    renderPage();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true, writable: true });
    await screen.findByRole("log");
    await user.click(screen.getByRole("button", { name: "Copy log" }));
    expect(writeText).toHaveBeenCalledTimes(1);
    const text = writeText.mock.calls[0]?.[0] as string;
    expect(text.split("\n")).toHaveLength(2);
    expect(text).toMatch(/^\d{2}:\d{2}:\d{2}\.\d{3} {2}Cloning into 'api'\.\.\.\n/);
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Log copied"));
  });

  it("downloads the log as deploy-<id>.log", async () => {
    mockApi(deploy({}), lines);
    globalThis.URL.createObjectURL = vi.fn(() => "blob:log");
    globalThis.URL.revokeObjectURL = vi.fn();
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    renderPage();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    await screen.findByRole("log");
    await user.click(screen.getByRole("button", { name: "Download log" }));
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
    expect(click).toHaveBeenCalledTimes(1);
    const anchor = click.mock.instances[0] as HTMLAnchorElement;
    expect(anchor.download).toBe("deploy-d-1.log");
    expect(anchor.href).toBe("blob:log");
    click.mockRestore();
  });

  it("shows the waiting empty state while nothing has been logged yet", async () => {
    mockApi(deploy({ status: "pending" }), []);
    renderPage();
    expect(await screen.findByText("Waiting for the runner to pick this up…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy log" })).toBeDisabled();
  });

  it("shows the no-output empty state for a terminal deploy without lines", async () => {
    mockApi(deploy({ status: "failed" }), []);
    renderPage();
    expect(await screen.findByText("No output was recorded.")).toBeInTheDocument();
  });

  it("renders the error display when the deploy cannot be loaded", async () => {
    mocks.get.mockRejectedValue(new Error("nope"));
    renderPage();
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });
});
