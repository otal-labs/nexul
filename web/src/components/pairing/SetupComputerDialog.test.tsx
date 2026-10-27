import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupComputerDialog } from "@/components/pairing/SetupComputerDialog";
import { Button } from "@/components/ui/button";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import type { Computer, ComputerSetup, HarnessProject, HarnessProvider, PairingDefaults, SetupTurnKind, SetupTurnState } from "@/models/Pairing";
import type { ActivityKind } from "@/models/Trail";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const created = {
  id: "c1",
  name: "Work laptop",
  server_url: "http://c1.nexul-computer.invalid",
  token_expires_at: "0001-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "",
  created_at: "2026-09-24T00:00:00Z",
  position: 0,
  runs_at_once: 1,
  running: 0,
  chat_project_id: "",
  skills_outdated: false,
  updated_at: "2026-09-24T00:00:00Z",
};

const apiError = (body: Record<string, string>) => Object.assign(new Error("request failed"), { response: { data: body } });

const emptySetup: ComputerSetup = { computer_id: "c1", confirmed_at: null, providers: [], skipped_providers: [], models: {}, model_options: {}, folder: "", turns: [] };

const renderDialog = (computer: Computer) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SetupComputerDialog computer={computer} trigger={<Button type="button">Open</Button>} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

describe("SetupComputerDialog", () => {
  const effort = {
    id: "effort",
    label: "Reasoning",
    type: "select" as const,
    choices: [
      { id: "medium", label: "Medium", is_default: true },
      { id: "high", label: "High" },
    ],
  };
  const paired: Computer = { ...created, token_expires_at: "2026-10-24T00:00:00Z", harness_version: "0.0.40" };
  const turn = (provider: string, name: string, state: SetupTurnState, status: string, run_id = "r0", turn_id = `t-${provider}`) => ({
    run_id,
    turn_id,
    provider,
    provider_name: name,
    kind: "setup" as SetupTurnKind,
    state,
    status,
    updated_at: "2026-09-24T00:00:00Z",
  });
  const run = (...names: [string, string][]) => ({ data: { run_id: "r1", computer_id: "c1", providers: names.map(([provider, name]) => ({ provider, name })) } });
  let setup: ComputerSetup;
  let providers: HarnessProvider[];
  let defaults: PairingDefaults;
  let chatFolder: string;
  let projects: HarnessProject[];

  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.put.mockReset();
    useSetupActivityStore.setState({ steps: {} });
    useSetupDraftStore.setState({ drafts: {} });
    setup = emptySetup;
    providers = [
      { id: "codex", driver: "codex", name: "Codex", needs_setup: true, skipped: false, models: [{ slug: "gpt", name: "GPT", is_default: true }] },
      { id: "claude", driver: "claudeAgent", name: "Claude", needs_setup: true, skipped: false, models: [{ slug: "claude-big", name: "Big", options: [effort] }, { slug: "claude-small", name: "Small", is_default: true }] },
      { id: "opencode", driver: "opencode", name: "OpenCode", needs_setup: true, skipped: false, models: [{ slug: "pickle", name: "Pickle", is_default: true }] },
    ];
    defaults = { placement: "in_order", clone_automatically: false };
    chatFolder = "";
    projects = [];
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/setup")) return { data: setup };
      if (url.endsWith("/projects")) return { data: { projects } };
      if (url.endsWith("/providers")) return { data: { providers } };
      if (url.endsWith("/defaults")) return { data: defaults };
      if (url === "/api/pairing/computers") return { data: { computers: [{ ...paired, chat_project_id: chatFolder }] } };
      return { data: { computers: [] } };
    });
    mocks.put.mockImplementation(async (_url: string, body: Partial<ComputerSetup>) => {
      setup = { ...setup, ...body };
      return { data: setup };
    });
  });

  it("opens the failed provider's row on its reason, and retries only that one", async () => {
    setup = {
      ...emptySetup,
      confirmed_at: "2026-09-24T00:00:00Z",
      turns: [turn("codex", "Codex", "confirmed", "Confirmed with 12 skills"), turn("opencode", "OpenCode", "failed", "No result within 10m0s")],
    };
    mocks.post.mockResolvedValue(run(["opencode", "OpenCode"]));
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("heading", { name: /set up work laptop/i })).toBeInTheDocument();
    const list = within(await screen.findByRole("list", { name: "Providers" }));
    const failedRow = await list.findByRole("button", { name: /^opencode/i });
    expect(failedRow).toHaveAttribute("aria-expanded", "true");
    expect(failedRow).toHaveTextContent(/No result within 10m0s/);
    expect(list.getByRole("button", { name: /^codex/i })).toHaveAttribute("aria-expanded", "false");
    expect(list.getByRole("button", { name: /^claude/i })).toHaveTextContent("Not set up yet");
    expect(screen.getByText("1/3 confirmed")).toBeInTheDocument();

    await user.click(list.getByRole("button", { name: /^retry opencode$/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/opencode/retry", { model: "pickle", model_options: [], folder: "" }));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("retries a failed skills update as another update, not a setup of that provider", async () => {
    setup = {
      ...emptySetup,
      confirmed_at: "2026-09-24T00:00:00Z",
      turns: [turn("claudeagent", "Claude", "confirmed", "Confirmed with 12 skills"), { ...turn("codex", "Codex", "failed", "permission denied"), kind: "skills" }],
    };
    mocks.post.mockResolvedValue(run(["codex", "Codex"]));
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    await user.click(await screen.findByRole("button", { name: /retry codex/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/skills"));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("sets every switched-on provider up at once and follows the running one's transcript until another is clicked", async () => {
    mocks.post.mockImplementation(async () => {
      setup = {
        ...emptySetup,
        turns: [
          turn("claudeagent", "Claude", "running", "Checking the skills Claude discovered", "r1", "t1"),
          turn("codex", "Codex", "running", "Connecting Nexul and installing skills", "r1", "t2"),
        ],
      };
      return run(["codex", "Codex"], ["claudeagent", "Claude"], ["opencode", "OpenCode"]);
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    await user.click(await screen.findByRole("button", { name: /set up all/i }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", {
        models: { codex: "gpt", claudeagent: "claude-small", opencode: "pickle" },
        model_options: { codex: [], claudeagent: [], opencode: [] },
        folder: "",
        providers: ["codex", "claudeagent", "opencode"],
      }),
    );
    expect(await screen.findByText("Setting up 3 providers")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /set up all/i })).toBeDisabled();
    expect(screen.getByRole("switch", { name: "Use Codex" })).toBeDisabled();
    expect(screen.getByRole("button", { name: /^codex/i })).toHaveAttribute("aria-expanded", "true");

    const push = (kind: ActivityKind, summary: string, call_id: string) =>
      act(() => useSetupActivityStore.getState().push("t2", { kind, call_id, tool: "Shell", summary, detail: "", at: "" }));
    push("tool_call", "nexul mcp add", "call-1");
    push("tool_result", "nexul mcp add", "call-1");
    push("tool_call", "ls ~/.claude/skills", "call-2");
    const log = screen.getByRole("log", { name: "Codex steps" });
    expect(within(log).getAllByText(/^(nexul mcp add|ls ~\/\.claude\/skills)$/).map((l) => l.textContent)).toEqual(["nexul mcp add", "ls ~/.claude/skills"]);

    await user.click(screen.getByRole("button", { name: /^opencode/i }));
    expect(screen.getByRole("button", { name: /^opencode/i })).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Starting its turn.")).toBeInTheDocument();
    expect(screen.queryByRole("log", { name: "Codex steps" })).not.toBeInTheDocument();
  });

  it("runs one provider on its own row while another still runs", async () => {
    setup = { ...emptySetup, turns: [turn("codex", "Codex", "running", "Connecting Nexul and installing skills", "r0", "t0")] };
    mocks.post.mockResolvedValue(run(["claudeagent", "Claude"]));
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    await user.click(await screen.findByRole("button", { name: "Set up Claude" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/claudeagent/retry", { model: "claude-small", model_options: [], folder: "" }));
    expect(screen.getByRole("button", { name: "Set up OpenCode" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: /set up codex/i })).not.toBeInTheDocument();
  });

  it("picks a model and its options per provider, preselected from the defaults, and sends them with Set up all and Retry", async () => {
    providers = [
      { id: "claude", driver: "claudeAgent", name: "Claude", needs_setup: true, skipped: false, models: [{ slug: "claude-big", name: "Big", options: [effort] }, { slug: "claude-small", name: "Small", is_default: true }] },
      { id: "opencode", driver: "opencode", name: "OpenCode", needs_setup: true, skipped: false, models: [{ slug: "pickle", name: "Pickle", is_default: true }, { slug: "gpt", name: "GPT", options: [effort] }] },
    ];
    defaults = { provider: "claude", model: "claude-big", placement: "in_order", clone_automatically: false };
    mocks.post.mockImplementation(async () => {
      setup = { ...emptySetup, turns: [turn("claudeagent", "Claude", "confirmed", "Confirmed with 12 skills", "r1", "t2"), turn("opencode", "OpenCode", "failed", "npx: command not found", "r1", "t3")] };
      return run(["claudeagent", "Claude"], ["opencode", "OpenCode"]);
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("combobox", { name: "Claude model" })).toHaveTextContent("Big");
    expect(screen.getByRole("combobox", { name: "OpenCode model" })).toHaveTextContent("Pickle");
    expect(screen.queryByRole("button", { name: "OpenCode model options" })).not.toBeInTheDocument();
    await pickOption(user, "OpenCode model", "Provider default");
    await user.click(screen.getByRole("button", { name: "Claude model options" }));
    await user.click(await screen.findByRole("menuitemradio", { name: "High" }));
    expect(screen.getByRole("button", { name: "Claude model options" })).toHaveTextContent("High");
    await user.click(screen.getByRole("button", { name: /set up all/i }));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", {
        models: { claudeagent: "claude-big", opencode: "" },
        model_options: { claudeagent: [{ id: "effort", value: "high" }], opencode: [] },
        folder: "",
        providers: ["claudeagent", "opencode"],
      }),
    );

    await pickOption(user, "OpenCode model", "GPT");
    await user.click(await screen.findByRole("button", { name: /^retry opencode$/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/opencode/retry", { model: "gpt", model_options: [], folder: "" }));
  });

  describe("switching providers off", () => {
    it("saves a switch at once, leaves the provider out of Set up all, and counts confirmations over the rest", async () => {
      setup = { ...emptySetup, turns: [turn("codex", "Codex", "confirmed", "Confirmed with 12 skills", "r0", "t0")] };
      mocks.post.mockResolvedValue(run(["claudeagent", "Claude"], ["opencode", "OpenCode"]));
      const user = userEvent.setup();
      renderDialog(paired);

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await user.click(await screen.findByRole("switch", { name: "Use Codex" }));
      await waitFor(() =>
        expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/choices", {
          skipped_providers: ["codex"],
          models: { codex: "gpt", claudeagent: "claude-small", opencode: "pickle" },
          model_options: { codex: [], claudeagent: [], opencode: [] },
          folder: "",
        }),
      );
      expect(screen.getByRole("button", { name: /^codex/i })).toHaveTextContent("Off · plays leave it out");
      expect(screen.queryByRole("combobox", { name: "Codex model" })).not.toBeInTheDocument();
      expect(screen.getByText("0/2 confirmed")).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: /set up all/i }));
      await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", expect.objectContaining({ providers: ["claudeagent", "opencode"] })));
    });

    it("opens with saved switches off, and needs one provider on to set anything up", async () => {
      setup = { ...emptySetup, skipped_providers: ["codex"] };
      const user = userEvent.setup();
      renderDialog(paired);

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("switch", { name: "Use Codex" })).not.toBeChecked();
      expect(screen.getByRole("switch", { name: "Use Claude" })).toBeChecked();
      expect(screen.getByRole("button", { name: /set up all/i })).toBeEnabled();

      await user.click(screen.getByRole("switch", { name: "Use Claude" }));
      await user.click(screen.getByRole("switch", { name: "Use OpenCode" }));
      expect(await screen.findByText(/switch on a provider to set it up/i)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /set up all/i })).toBeDisabled();
      expect(mocks.post).not.toHaveBeenCalled();
    });

    it("puts the switch back when the save is refused", async () => {
      mocks.put.mockRejectedValueOnce(apiError({ message: "nope" }));
      const user = userEvent.setup();
      renderDialog(paired);

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await user.click(await screen.findByRole("switch", { name: "Use Codex" }));
      await waitFor(() => expect(screen.getByRole("switch", { name: "Use Codex" })).toBeChecked());
    });
  });

  it("runs setup in the computer's chat folder, or the one picked, and retries in it too", async () => {
    providers = [providers[0]!];
    projects = [
      { id: "t3-gone", title: "StreamerBotChat", path: "/home/me/StreamerBotChat" },
      { id: "t3-app", title: "App", path: "/home/me/app" },
    ];
    chatFolder = "t3-gone";
    mocks.post.mockImplementation(async () => {
      setup = { ...emptySetup, turns: [turn("codex", "Codex", "failed", "workspace folder no longer exists", "r1", "t2")] };
      return run(["codex", "Codex"]);
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("combobox", { name: "Runs in" })).toHaveTextContent("StreamerBotChat");
    await user.click(screen.getByRole("button", { name: /set up all/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", expect.objectContaining({ folder: "/home/me/StreamerBotChat" })));

    await pickOption(user, "Runs in", /^App/);
    expect(screen.getByText("/home/me/app")).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /^retry codex$/i }));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/codex/retry", { model: "gpt", model_options: [], folder: "/home/me/app" }),
    );
  });

  describe("closing", () => {
    beforeEach(() => {
      projects = [{ id: "t3-app", title: "App", path: "/home/me/app" }];
    });

    it("stays open on a click outside it", async () => {
      // The modal sets pointer-events none on the page behind it; the click still has to reach the outside handler.
      const user = userEvent.setup({ pointerEventsCheck: 0 });
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await screen.findByRole("switch", { name: "Use Codex" });

      await user.click(document.body);
      expect(screen.getByRole("heading", { name: /set up work laptop/i })).toBeInTheDocument();
    });

    it("Cancel drops an unsaved model, so the next open shows what was saved", async () => {
      const user = userEvent.setup();
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await pickOption(user, "Claude model", "Big");

      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(screen.queryByRole("heading", { name: /set up work laptop/i })).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("combobox", { name: "Claude model" })).toHaveTextContent("Small");
      expect(mocks.put).not.toHaveBeenCalled();
    });

    it("Done saves the models, options, and folder, and the next open shows them", async () => {
      const user = userEvent.setup();
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await pickOption(user, "Claude model", "Big");
      await user.click(screen.getByRole("button", { name: "Claude model options" }));
      await user.click(await screen.findByRole("menuitemradio", { name: "High" }));

      await user.click(screen.getByRole("button", { name: "Done" }));
      await waitFor(() =>
        expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/choices", {
          skipped_providers: [],
          models: { codex: "gpt", claudeagent: "claude-big", opencode: "pickle" },
          model_options: { codex: [], claudeagent: [{ id: "effort", value: "high" }], opencode: [] },
          folder: "/home/me/app",
        }),
      );
      await waitFor(() => expect(screen.queryByRole("heading", { name: /set up work laptop/i })).not.toBeInTheDocument());

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("combobox", { name: "Claude model" })).toHaveTextContent("Big");
      expect(screen.getByRole("button", { name: "Claude model options" })).toHaveTextContent("High");
      expect(mocks.post).not.toHaveBeenCalled();
    });
  });
});
