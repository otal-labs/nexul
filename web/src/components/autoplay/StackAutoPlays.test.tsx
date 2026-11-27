import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";
import { pickOption } from "@/test/pickOption";
import { renderAgentPage } from "@/test/renderAgentPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: mocks,
  errorMessage: (error: unknown) => (error as Error).message,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play = (overrides: Partial<Play> = {}): Play => ({
  id: "pl-1",
  workspace_id: "ws-1",
  label: "Investigate a crash",
  type: "stack",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: null,
  excluded_project_ids: [],
  repository: "app",
  builtin_key: "investigate-crash",
  created_by: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const autoPlay = (overrides: Partial<AutoPlay> = {}): AutoPlay => ({
  id: "ap-1",
  play_id: "pl-1",
  workspace_id: "ws-1",
  enabled: false,
  moment: "stack.service_crashed",
  moment_stage: null,
  schedule: null,
  one_at_a_time: false,
  resource: "",
  conditions: { match: "all", groups: [] },
  priority: { rules: [], otherwise: "normal" },
  once_within_minutes: 0,
  run_on: "person",
  run_as: "",
  memory_ids: [],
  created_by: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const WRITE = ["plays:read", "plays:write", "autoplays:read", "autoplays:write", "autoplays:delete"];
let current: Play = play();
let autoPlays: AutoPlay[] = [];

const serve = () =>
  mocks.get.mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    const projectId = (config?.params as { project_id?: string } | undefined)?.project_id;
    if (url === "/api/workspaces/ws-1/plays") return { data: [current] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: WRITE } };
    if (url === "/api/workspaces/ws-1/plays/pl-1/auto-plays") return { data: autoPlays };
    if (url === "/api/workspaces/ws-1/plays/pl-1/queued") return { data: [] };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Website" }] };
    if (url === "/api/memories") return { data: [] };
    if (url === "/api/stacks") return { data: [{ id: "st-1", name: "acme-web", project_id: projectId ?? "p-1" }] };
    if (url === "/api/stacks/st-1/services") return { data: [{ id: "sv-1", stack_id: "st-1", name: "api" }, { id: "sv-2", stack_id: "st-1", name: "worker" }] };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-bob", login: "bob", display_name: "Bob", avatar_url: "" }] } };
    if (url === "/api/auth/me") return { data: { user: { id: "u-me" } } };
    if (url === "/api/pairing/computers") return { data: { computers: [] } };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url.startsWith("/api/permissions")) return { data: { grants: [] } };
    return { data: [] };
  });

const openAutoPlays = async () => {
  const user = userEvent.setup();
  renderAgentPage(`/acme/agent/${current.type}`);
  await screen.findByRole("button", { name: ADD });
  return { user };
};

const ADD = "+ Add auto play";

const footer = (name: string) => screen.getAllByRole("button", { name }).at(-1)!;

beforeEach(() => {
  current = play();
  autoPlays = [];
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  serve();
});

describe("a stack play's auto plays", () => {
  it("names whose computers a stack auto play runs on, or that nobody is picked", async () => {
    autoPlays = [autoPlay(), autoPlay({ id: "ap-3", moment: "stack.error_rate_crossed", run_as: "u-bob", enabled: true })];
    await openAutoPlays();

    const rows = await screen.findAllByRole("button", { name: /When a stack/ });
    await waitFor(() => expect(rows[0]).toHaveTextContent("When a stack service crashes Needs a person"));
    await waitFor(() => expect(rows[1]).toHaveTextContent("Bob's computers"));
  });

  it("offers the stack moments and conditions, runs as you by default, and saves that", async () => {
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: ADD }));

    expect(screen.getByRole("combobox", { name: "Moment" })).toHaveTextContent("Service crashes");
    await user.click(screen.getByRole("combobox", { name: "Moment" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Service crashes", "Service recovers", "Error rate goes over its limit"]);
    await user.keyboard("{Escape}");

    const runOn = screen.getByRole("region", { name: "Runs as" });
    expect(within(runOn).getByRole("combobox", { name: "Runs as" })).toHaveTextContent("A person");
    expect(await within(runOn).findByText("Runs as you, on your computers in your order.")).toBeInTheDocument();
    expect(within(runOn).queryByRole("button", { name: "Run as me" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "+ Add a condition" }));
    await user.click(screen.getByRole("combobox", { name: "Field" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Project", "Stack", "Service"]);
    await user.click(screen.getByRole("option", { name: "Service" }));
    await user.click(screen.getByRole("button", { name: "Service values" }));
    await user.click(await screen.findByRole("checkbox", { name: "api" }));
    await user.keyboard("{Escape}");

    mocks.post.mockResolvedValue({ data: autoPlay({ id: "ap-2" }) });
    await user.click(footer("Save"));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays", {
        enabled: false,
        moment: "stack.service_crashed",
        moment_stage: null,
        schedule: null,
        conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "service", op: "is", values: ["api"] }] }] },
        priority: { rules: [], otherwise: "normal" },
        once_within_minutes: 60,
        run_on: "person",
        run_as_me: true,
        memory_ids: [],
        one_at_a_time: false,
        resource: "",
        computer_rule: { labels_in: [], labels_out: [], ids: [] },
      }),
    );
  });

  it("offers a stack play only the person run-on", async () => {
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: ADD }));

    await user.click(within(screen.getByRole("region", { name: "Runs as" })).getByRole("combobox", { name: "Runs as" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["A person"]);
    await user.keyboard("{Escape}");
  });

  it("says who a saved auto play runs as, with Run as me only when it is someone else, and keeps them through a save", async () => {
    autoPlays = [autoPlay({ run_as: "u-bob" })];
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /When a stack/ }));

    const runOn = screen.getByRole("region", { name: "Runs as" });
    expect(await within(runOn).findByText("Runs as Bob, on their computers.")).toBeInTheDocument();

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer("Save"));
    await waitFor(() => expect(mocks.patch).toHaveBeenCalledTimes(1));
    expect(mocks.patch.mock.calls[0]?.[1]).toMatchObject({ run_on: "person" });
    expect(mocks.patch.mock.calls[0]?.[1]).not.toHaveProperty("run_as_me");
  });

  it("makes the viewer the person with Run as me, and saves run_as_me", async () => {
    autoPlays = [autoPlay({ run_as: "u-bob" })];
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /When a stack/ }));

    const runOn = screen.getByRole("region", { name: "Runs as" });
    await user.click(await within(runOn).findByRole("button", { name: "Run as me" }));
    expect(within(runOn).getByText("Runs as you, on your computers in your order.")).toBeInTheDocument();
    expect(within(runOn).queryByRole("button", { name: "Run as me" })).not.toBeInTheDocument();

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer("Save"));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1", expect.objectContaining({ run_on: "person", run_as_me: true })),
    );
  });

  it("reads a seeded auto play nobody is picked for as nobody picked, until Run as me picks", async () => {
    autoPlays = [autoPlay()];
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /When a stack/ }));

    const runOn = screen.getByRole("region", { name: "Runs as" });
    expect(await within(runOn).findByRole("combobox", { name: "Runs as" })).toHaveTextContent("Pick who it runs as");

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer("Save"));
    await waitFor(() => expect(mocks.patch).toHaveBeenCalledTimes(1));
    expect(mocks.patch.mock.calls[0]?.[1]).not.toHaveProperty("run_as_me");

    await user.click(await screen.findByRole("button", { name: /When a stack/ }));
    await user.click(await screen.findByRole("button", { name: "Run as me" }));
    await user.click(footer("Save"));
    await waitFor(() => expect(mocks.patch).toHaveBeenCalledTimes(2));
    expect(mocks.patch.mock.calls[1]?.[1]).toMatchObject({ run_on: "person", run_as_me: true });
  });

  it("switches an auto play on with its person as it is, so the server still refuses an unpicked one", async () => {
    autoPlays = [autoPlay(), autoPlay({ id: "ap-2", moment: "stack.service_recovered", run_as: "u-me", enabled: true })];
    const { user } = await openAutoPlays();
    mocks.patch.mockResolvedValue({ data: autoPlays[0] });

    await user.click(await screen.findByRole("switch", { name: "Turn When a stack service recovers on or off" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-2", expect.objectContaining({ enabled: false, run_on: "person" }));

    await user.click(screen.getByRole("switch", { name: "Turn When a stack service crashes on or off" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1", expect.objectContaining({ enabled: true }));
    expect(mocks.patch.mock.calls.at(-1)?.[1]).not.toHaveProperty("run_as_me");
  });
});

describe("the run-on choices of the other play types", () => {
  it("are developer, tester, whoever caused it and a person for a ticket play", async () => {
    current = play({ label: "Fix with AI", type: "ticket", show_when_stage: "progress", builtin_key: "" });
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: ADD }));

    const runOn = screen.getByRole("region", { name: "Runs as" });
    await user.click(within(runOn).getByRole("combobox", { name: "Runs as" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["The ticket's developer", "The ticket's tester", "Whoever moved the ticket", "A person"]);
    await user.keyboard("{Escape}");
    await pickOption(user, "Runs as", "A person");
    expect(await within(runOn).findByText("Runs as you, on your computers in your order.")).toBeInTheDocument();
  });

  it("are whoever caused it and a person for a doc play", async () => {
    current = play({ label: "To tickets via AI", type: "doc", builtin_key: "" });
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: ADD }));

    const runOn = screen.getByRole("region", { name: "Runs as" });
    expect(within(runOn).getByRole("combobox", { name: "Runs as" })).toHaveTextContent("Whoever created the doc");
    await user.click(within(runOn).getByRole("combobox", { name: "Runs as" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Whoever created the doc", "A person"]);
    await user.keyboard("{Escape}");
  });
});
