import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MachineMenu } from "@/components/runner/MachineMenu";
import { getMachineCommandsKey, machineCommandFollower } from "@/hooks/MachineCommandHooks";
import type { ComputerCommand } from "@/models/ComputerCommand";
import type { Machine } from "@/models/Runner";
import { followFrame } from "@/test/followFrame";
import { resetViewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), useContainerLogs: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post, put: mocks.put }, errorMessage: (e: Error) => e.message }));
vi.mock("@/hooks/ContainerLogHooks", () => ({ useContainerLogs: mocks.useContainerLogs }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);
vi.mock("@/hooks/useWorkspacePath", () => ({ useWorkspacePath: () => (path: string) => path }));
vi.mock("react-router", async (original) => ({ ...(await original<typeof import("react-router")>()), useNavigate: () => vi.fn() }));

const prod = (overrides: Partial<Machine> = {}): Machine => ({
  id: "m-1",
  name: "prod",
  stack_root: "/data/nexul",
  first_seen: "",
  last_seen: "",
  actions: ["machines:see", "machines:shell"],
  shell_enabled: true,
  ...overrides,
});

const command = (overrides: Partial<ComputerCommand> = {}): ComputerCommand => ({
  id: "j1",
  runner_id: "r-edge",
  machine_id: "m-1",
  runner_name: "edge",
  requester_id: "u-alice",
  requester_name: "alice",
  adapter: "web",
  shell: "",
  command: "df -h",
  cwd: "",
  timeout_seconds: 60,
  status: "running",
  exit_code: null,
  error: "",
  output_truncated: false,
  output_purged: false,
  created_at: new Date().toISOString(),
  started_at: null,
  ended_at: null,
  ...overrides,
});

const runners = [
  { id: "r-edge", name: "edge", connected: true, last_seen: "", running_job: null, version: "v1", machine: "prod" },
  { id: "r-other", name: "other", connected: true, last_seen: "", running_job: null, version: "v1", machine: "staging" },
];

const routeGet = (path: string) => {
  if (path === "/api/runners") return Promise.resolve({ data: runners });
  return Promise.resolve({ data: { items: [command()], total: 1 } });
};

const renderMenu = (machine: Machine) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MachineMenu machine={machine} menu={{}} manages={false} addsRunners={false} />
    </QueryClientProvider>,
  );
  return client;
};

const openCommands = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: "More actions for prod" }));
  await user.click(await screen.findByRole("menuitem", { name: /Commands/ }));
};

describe("MachineCommands", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetViewer();
    mocks.get.mockImplementation(routeGet);
    mocks.post.mockResolvedValue({ data: command({ id: "j2" }) });
    mocks.put.mockResolvedValue({ data: prod({ shell_enabled: false }) });
    mocks.useContainerLogs.mockReturnValue({ lines: [{ seq: 0, ts: "", stream: "stdout", text: "/dev/sda1 40G" }], status: "live", reason: undefined });
  });

  it("offers no Commands to a viewer without Run commands on the machine", () => {
    renderMenu(prod({ actions: ["machines:see", "machines:deploy"] }));
    expect(screen.queryByRole("button", { name: "More actions for prod" })).not.toBeInTheDocument();
  });

  it("lists who ran what on which runner, runs a command on the machine's runner and stops a running one", async () => {
    const user = userEvent.setup();
    renderMenu(prod());
    await openCommands(user);

    expect(await screen.findByText("df -h")).toBeInTheDocument();
    expect(screen.getByText("edge")).toBeInTheDocument();
    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith("/api/machines/m-1/commands", { params: { limit: 10 } });

    await user.type(screen.getByRole("textbox", { name: "Command" }), "uptime");
    await user.click(screen.getByRole("button", { name: "Run" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/runners/r-edge/commands", { command: "uptime" });

    await user.click(screen.getByRole("button", { name: "Stop df -h" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/runners/commands/j1/cancel");

    await user.click(screen.getByRole("button", { name: "Output of df -h" }));
    expect(await screen.findByText("/dev/sda1 40G")).toBeInTheDocument();
    expect(mocks.useContainerLogs).toHaveBeenCalledWith("/runners/commands/j1/output");
  });

  it("offers no command box while the machine's switch is off, and turns it on", async () => {
    const user = userEvent.setup();
    renderMenu(prod({ shell_enabled: false }));
    await openCommands(user);

    expect(await screen.findByText("df -h")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Command" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("switch", { name: "Run commands from Nexul" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/machines/m-1/shell", { enabled: true });
  });

  it("follows a command's changes on its machine's list alone", async () => {
    const client = new QueryClient();
    client.setQueryData([getMachineCommandsKey, "m-1"], [command()]);
    client.setQueryData([getMachineCommandsKey, "m-2"], [command({ machine_id: "m-2" })]);
    await followFrame(machineCommandFollower, "runner.shell_job_changed", { machine_id: "m-1" }, client);
    expect(client.getQueryState([getMachineCommandsKey, "m-1"])?.isInvalidated).toBe(true);
    expect(client.getQueryState([getMachineCommandsKey, "m-2"])?.isInvalidated).toBe(false);
  });
});
