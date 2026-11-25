import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputerCommands } from "@/components/settings/ComputerCommands";
import { computerCommandFollower, getComputerCommandsKey } from "@/hooks/ComputerCommandHooks";
import type { ComputerCommand } from "@/models/ComputerCommand";
import type { Computer } from "@/models/Pairing";
import { followFrame } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), useContainerLogs: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post, patch: mocks.patch }, errorMessage: (e: Error) => e.message }));
vi.mock("@/hooks/ContainerLogHooks", () => ({ useContainerLogs: mocks.useContainerLogs }));

const laptop = (overrides: Partial<Computer> = {}): Computer =>
  ({
    id: "c-laptop",
    name: "Laptop",
    shell_enabled: true,
    runner: { connected: true },
    facts: { t3: { state: "answering" }, providers: [], projects: [], shell: { enabled: true } },
    ...overrides,
  }) as Computer;

const command = (overrides: Partial<ComputerCommand>): ComputerCommand => ({
  id: "j1",
  computer_id: "c-laptop",
  owner_id: "u-alice",
  requester_id: "u-alice",
  adapter: "mcp",
  shell: "",
  command: "go test ./...",
  cwd: "",
  timeout_seconds: 60,
  status: "succeeded",
  exit_code: 0,
  error: "",
  output_truncated: false,
  output_purged: false,
  created_at: new Date().toISOString(),
  started_at: null,
  ended_at: null,
  ...overrides,
});

const renderCommands = (computer: Computer, client = new QueryClient({ defaultOptions: { queries: { retry: false } } })) => {
  render(
    <QueryClientProvider client={client}>
      <ComputerCommands computer={computer} />
    </QueryClientProvider>,
  );
  return client;
};

describe("ComputerCommands", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useContainerLogs.mockReturnValue({ lines: [{ seq: 0, ts: "", stream: "stdout", text: "ok app 0.4s" }], status: "live", reason: undefined });
    mocks.get.mockResolvedValue({ data: { items: [command({}), command({ id: "j2", command: "make lint", status: "failed", exit_code: 2, adapter: "web" })], total: 2 } });
  });

  it("lists the newest commands with how each ended and opens one's output", async () => {
    const user = userEvent.setup();
    renderCommands(laptop());

    expect(await screen.findByText("go test ./...")).toBeInTheDocument();
    expect(screen.getByText("exit 2")).toBeInTheDocument();
    expect(screen.getByText("agent")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/pairing/computers/c-laptop/commands", { params: { limit: 10 } });

    await user.click(screen.getByRole("button", { name: "Output of go test ./..." }));
    expect(await screen.findByText("ok app 0.4s")).toBeInTheDocument();
    expect(mocks.useContainerLogs).toHaveBeenCalledWith("/pairing/commands/j1/output");
  });

  it("runs a command typed in the box and stops a running one", async () => {
    const user = userEvent.setup();
    mocks.get.mockResolvedValue({ data: { items: [command({ status: "running", exit_code: null })], total: 1 } });
    mocks.post.mockResolvedValue({ data: command({ id: "j3", status: "queued" }) });
    renderCommands(laptop());

    await user.type(screen.getByRole("textbox", { name: "Command" }), "uname -a{Enter}");
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c-laptop/commands", { command: "uname -a" });

    await user.click(await screen.findByRole("button", { name: "Stop go test ./..." }));
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/commands/j1/cancel");
  });

  it("turns commands off with the switch, and says when the computer itself turned them off", async () => {
    const user = userEvent.setup();
    mocks.patch.mockResolvedValue({ data: laptop({ shell_enabled: false }) });
    renderCommands(laptop({ facts: { t3: { state: "answering" }, providers: [], projects: [], shell: { enabled: false } } }));

    expect(screen.getByText("Off on the computer")).toBeInTheDocument();
    await user.click(screen.getByRole("switch"));
    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { shell_enabled: false });
  });

  it("hides the box while its owner keeps commands off", async () => {
    renderCommands(laptop({ shell_enabled: false }));
    expect(await screen.findByText("go test ./...")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Command" })).not.toBeInTheDocument();
  });

  it("refreshes the list when a command on this computer changes", async () => {
    const client = renderCommands(laptop());
    await screen.findByText("go test ./...");
    await act(() => followFrame(computerCommandFollower, "computer.shell_job_changed", { computer_id: "c-laptop", status: "running" }, client));
    expect(mocks.get).toHaveBeenCalledTimes(2);
    expect(client.getQueryData([getComputerCommandsKey, "c-laptop"])).toBeDefined();
  });
});

describe("ComputerCommands on a computer shared with the viewer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.get.mockResolvedValue({ data: { items: [command({ requester_id: "u-dave", requester_name: "dave", adapter: "web" })], total: 1 } });
  });

  it("offers the box but never the owner's switch", async () => {
    renderCommands(laptop({ shell_enabled: false, shared: { owner_id: "u-alice", owner_name: "alice", permissions: ["computers:see", "computers:run_commands"] } }));
    expect(await screen.findByRole("textbox", { name: "Command" })).toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
  });

  it("names who ran a command in its owner's list", async () => {
    renderCommands(laptop());
    expect(await screen.findByText("dave")).toBeInTheDocument();
  });
});
