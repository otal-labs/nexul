import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { Button } from "@/components/ui/button";
import { setCachedTunnelStatus } from "@/hooks/PairingHooks";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import type { Computer, ComputerSetup, HarnessProject, HarnessProvider, PairingDefaults, SetupTurnState } from "@/models/Pairing";
import { pickOption } from "@/test/pickOption";

const access = vi.hoisted(() => ({ sections: ["connectors"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useCanOpenSection: (section: string) => access.sections.includes(section) }));

beforeEach(() => {
  access.sections = ["connectors"];
});

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const created = {
  id: "c1",
  name: "Work laptop",
  server_url: "https://work-laptop-ab12cd34.example.com",
  token_expires_at: "0001-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "",
  created_at: "2026-09-24T00:00:00Z",
  updated_at: "2026-09-24T00:00:00Z",
  tunnel: { tunnel_id: "tun-1", hostname: "work-laptop-ab12cd34.example.com" },
};

const apiError = (body: Record<string, string>, errors?: Record<string, string[]>) =>
  Object.assign(new Error("request failed"), { response: { data: { ...body, ...(errors && { errors }) } } });

const emptySetup: ComputerSetup = { computer_id: "c1", confirmed_at: null, providers: [], skipped_providers: [], models: {}, model_options: {}, folder: "", turns: [] };

const renderDialog = (existing?: Computer) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PairComputerDialog existing={existing} trigger={<Button type="button">{existing ? "Open" : "Pair a computer"}</Button>} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

const nameTheComputer = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: /pair a computer/i }));
  await user.type(await screen.findByLabelText(/computer name/i), "Work laptop");
  await user.click(screen.getByRole("button", { name: /create tunnel/i }));
};

const reachPairStep = async (user: ReturnType<typeof userEvent.setup>, client: QueryClient) => {
  mocks.post.mockResolvedValueOnce({ data: created });
  await nameTheComputer(user);
  await screen.findByText(/waiting for connection/i);
  act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: true }));
  await user.click(await screen.findByRole("button", { name: /^next$/i }));
  await screen.findByLabelText(/one-time pairing token/i);
};

describe("PairComputerDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/tunnel/token")) return { data: { token: "eyJ-connector-token" } };
      if (url.endsWith("/tunnel/status")) return { data: { tunnel: "inactive", harness_reachable: false } };
      if (url.endsWith("/setup")) return { data: emptySetup };
      return { data: { computers: [] } };
    });
  });

  it("creates the tunnel, shows the commands, and unlocks Next once both checks pass", async () => {
    mocks.post.mockResolvedValue({ data: created });
    const user = userEvent.setup();
    const client = renderDialog();

    await nameTheComputer(user);
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/tunnel", { name: "Work laptop", port: 3773 }));

    expect(await screen.findByText(/tunnel\.sh \| sh -s -- eyJ-connector-token/)).toBeInTheDocument();
    expect(screen.getByText(/waiting for connection/i)).toBeInTheDocument();
    expect(screen.getByText("work-laptop-ab12cd34.example.com")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^next$/i })).toBeDisabled();

    await user.click(screen.getByRole("tab", { name: /windows/i }));
    expect(screen.getByText(/tunnel\.ps1\)\)\) eyJ-connector-token/)).toBeInTheDocument();

    act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: false }));
    expect(await screen.findByText("Online")).toBeInTheDocument();
    expect(screen.getByText(/start t3 code on this computer/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^next$/i })).toBeDisabled();

    act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" }));
    expect(await screen.findByText("Connected")).toBeInTheDocument();
    expect(screen.getByText("T3 Code 0.0.40")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^next$/i }));
    expect(await screen.findByLabelText(/one-time pairing token/i)).toBeInTheDocument();
  });

  it("pairs T3 Code over the verified hostname with only the token typed, then moves on to Set up", async () => {
    const user = userEvent.setup();
    const client = renderDialog();
    await reachPairStep(user, client);

    expect(screen.getByLabelText(/^name$/i)).toHaveValue("Work laptop");
    expect(screen.getByLabelText(/^name$/i)).toHaveAttribute("readonly");
    expect(screen.getByLabelText(/t3 server url/i)).toHaveValue("https://work-laptop-ab12cd34.example.com");
    expect(screen.getByLabelText(/t3 server url/i)).toHaveAttribute("readonly");
    expect(screen.getByText("t3 pair")).toBeInTheDocument();

    mocks.post.mockResolvedValueOnce({ data: { ...created, token_expires_at: "2026-10-24T00:00:00Z", harness_version: "0.0.40" } });
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenLastCalledWith("/api/pairing/computers/c1/pair", { token: "t3-pair-token" }));
    expect(await screen.findByRole("button", { name: /start setup/i })).toBeInTheDocument();
    expect(screen.getByText(/each provider on work laptop takes one short turn/i)).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /~\/\.claude\/skills\//i })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: /~\/\.agents\/skills\//i })).toBeChecked();
    expect(screen.getByRole("button", { name: /^done$/i })).toBeInTheDocument();
  });

  it("shows a refused token on the token field and an unreachable harness on the URL field", async () => {
    const user = userEvent.setup();
    const client = renderDialog();
    await reachPairStep(user, client);

    const refused = "the harness refused this token, run t3 pair for a fresh one";
    mocks.post.mockRejectedValueOnce(apiError({ message: refused, code: "INVALID" }, { token: [refused] }));
    await user.type(screen.getByLabelText(/one-time pairing token/i), "stale");
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(refused)).toBeInTheDocument();
    expect(screen.getByLabelText(/one-time pairing token/i)).toHaveAttribute("aria-invalid", "true");

    const unreachable = "couldn't reach the harness";
    mocks.post.mockRejectedValueOnce(apiError({ message: unreachable, code: "RETRYABLE" }, { server_url: [unreachable] }));
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(unreachable)).toBeInTheDocument();
    expect(screen.getByLabelText(/t3 server url/i)).toHaveAttribute("aria-invalid", "true");
    expect(screen.queryByText(/is paired/i)).not.toBeInTheDocument();
  });

  it("pairs a machine the server can already reach by URL, from Advanced", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    await user.click(screen.getByRole("button", { name: /pair by url/i }));

    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(/computer name is required/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText(/^name$/i), "VPS");
    await user.type(screen.getByLabelText(/t3 server url/i), "https://vps.example.com");
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    mocks.post.mockResolvedValueOnce({ data: { ...created, id: "c2", name: "VPS", tunnel: undefined } });
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers", {
        name: "VPS",
        server_url: "https://vps.example.com",
        token: "t3-pair-token",
      }),
    );
    expect(await screen.findByText(/each provider on vps takes one short turn/i)).toBeInTheDocument();
  });

  it("shows a failure with no field under the form", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    await user.click(screen.getByRole("button", { name: /pair by url/i }));
    await user.type(screen.getByLabelText(/^name$/i), "VPS");
    await user.type(screen.getByLabelText(/t3 server url/i), "https://vps.example.com");
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    mocks.post.mockRejectedValueOnce(apiError({ message: "internal error", code: "INTERNAL" }));
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("internal error");
  });

  it.each([
    ["zero_trust_disabled", /zero trust isn't enabled/i, /open zero trust/i],
    ["cloudflare_not_connected", /cloudflare isn't connected/i, /connect cloudflare/i],
  ])("explains a missing %s prerequisite with its fix and retries", async (reason, title, fix) => {
    mocks.post.mockRejectedValueOnce(apiError({ message: "missing", code: "INVALID", reason })).mockResolvedValueOnce({ data: created });
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(title);
    expect(screen.getByRole("link", { name: fix })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /try again/i }));
    expect(await screen.findByText(/waiting for connection/i)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(2);
  });

  it("shows any other failure inline under the form", async () => {
    mocks.post.mockRejectedValue(apiError({ message: "the instance's host is in no Cloudflare zone", code: "INVALID" }));
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/no cloudflare zone/i);
    expect(screen.queryByRole("button", { name: /try again/i })).not.toBeInTheDocument();
  });

  it("asks for a name and a valid port before creating anything", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    const port = screen.getByLabelText(/t3 code port/i);
    await user.clear(port);
    await user.type(port, "70000");
    await user.click(screen.getByRole("button", { name: /create tunnel/i }));

    expect(await screen.findByText(/name this computer/i)).toBeInTheDocument();
    expect(screen.getByText(/port between 1 and 65535/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("starts over at the name step after closing", async () => {
    mocks.post.mockResolvedValue({ data: created });
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    await screen.findByText(/waiting for connection/i);
    await user.click(screen.getByRole("button", { name: /close/i }));
    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    expect(await screen.findByLabelText(/computer name/i)).toBeInTheDocument();
  });

  it("resumes a computer still pairing at its tunnel, without naming it again", async () => {
    const user = userEvent.setup();
    const client = renderDialog(created);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("heading", { name: /pair work laptop/i })).toBeInTheDocument();
    expect(await screen.findByText(/waiting for connection/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/computer name/i)).not.toBeInTheDocument();

    act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: true }));
    await user.click(await screen.findByRole("button", { name: /^next$/i }));
    expect(await screen.findByLabelText(/one-time pairing token/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });
  it("explains a missing Cloudflare connection without a Settings link to a viewer who can't open Connectors", async () => {
    access.sections = [];
    mocks.post.mockRejectedValueOnce(apiError({ message: "missing", code: "INVALID", reason: "cloudflare_not_connected" }));
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/cloudflare isn't connected/i);
    expect(screen.queryByRole("link", { name: /connect cloudflare/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
  });

});

describe("PairComputerDialog opened at Set up", () => {
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
    state,
    status,
    updated_at: "2026-09-24T00:00:00Z",
  });
  let setup: ComputerSetup;
  let providers: HarnessProvider[];
  let defaults: PairingDefaults;
  let projects: HarnessProject[];

  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    useSetupActivityStore.setState({ steps: {} });
    useSetupDraftStore.setState({ drafts: {} });
    setup = emptySetup;
    providers = [];
    defaults = {};
    projects = [];
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/setup")) return { data: setup };
      if (url.endsWith("/projects")) return { data: { projects } };
      if (url.endsWith("/providers")) return { data: { providers } };
      if (url.endsWith("/defaults")) return { data: defaults };
      return { data: { computers: [] } };
    });
  });

  it("starts at Set up with the earlier steps locked, lists each provider's newest turn as a name-only row, and retries only the failed one", async () => {
    setup = {
      ...emptySetup,
      confirmed_at: "2026-09-24T00:00:00Z",
      turns: [turn("codex", "Codex", "confirmed", "Confirmed with 12 skills"), turn("opencode", "opencode", "failed", "No result within 10m0s")],
    };
    mocks.post.mockResolvedValue({ data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "opencode", name: "opencode" }] } });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("heading", { name: /set up work laptop/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /connect/i })).toBeDisabled();
    const list = within(await screen.findByRole("list", { name: "Providers" }));
    const failedRow = list.getByRole("button", { name: /opencode/i }).closest("li");
    expect(failedRow).toHaveTextContent(/^opencode: FailedRetry$/);
    expect(list.queryByText(/Confirmed with 12 skills/)).not.toBeInTheDocument();
    expect(screen.getByText(/No result within 10m0s/)).toBeInTheDocument();
    expect(screen.getByText("1/2 confirmed")).toBeInTheDocument();

    await user.click(list.getByRole("button", { name: /^retry$/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/opencode/retry", { model: "", model_options: [], folder: "" }));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("lists every provider of a started run at once and follows the running one's transcript until another is clicked", async () => {
    mocks.post.mockImplementation(async () => {
      setup = {
        ...emptySetup,
        turns: [
          turn("claudeagent", "Claude", "confirmed", "Confirmed with 12 skills", "r1", "t1"),
          turn("codex", "Codex", "running", "Connecting Nexul and installing skills", "r1", "t2"),
        ],
      };
      return {
        data: {
          run_id: "r1",
          computer_id: "c1",
          providers: [
            { provider: "claudeagent", name: "Claude" },
            { provider: "codex", name: "Codex" },
            { provider: "opencode", name: "OpenCode" },
          ],
        },
      };
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    await user.click(await screen.findByRole("button", { name: /start setup/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", { models: {}, model_options: {}, folder: "" }));
    expect(await screen.findByRole("heading", { name: "Codex" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /codex/i })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeDisabled();

    act(() => useSetupActivityStore.getState().push("t2", "nexul mcp add", "call-1", true));
    act(() => useSetupActivityStore.getState().push("t2", "nexul mcp add", "call-1"));
    act(() => useSetupActivityStore.getState().push("t2", "ls ~/.claude/skills", "call-2", true));
    const log = screen.getByRole("log", { name: "Codex steps" });
    expect(within(log).getAllByText(/nexul mcp add|ls ~\/\.claude\/skills/).map((l) => l.textContent)).toEqual(["nexul mcp add", "ls ~/.claude/skills"]);

    await user.click(screen.getByRole("button", { name: /opencode/i }));
    expect(screen.getByRole("heading", { name: "OpenCode" })).toBeInTheDocument();
    expect(screen.getByText("Waiting for its turn. Codex is setting up now.")).toBeInTheDocument();
    expect(screen.queryByRole("log", { name: "Codex steps" })).not.toBeInTheDocument();
  });

  it("picks a model and its options per provider, preselected from the defaults, and sends them with Start and Retry", async () => {
    providers = [
      { id: "claude", driver: "claudeAgent", name: "Claude", needs_setup: true, models: [{ slug: "claude-big", name: "Big", options: [effort] }, { slug: "claude-small", name: "Small", is_default: true }] },
      { id: "opencode", driver: "opencode", name: "OpenCode", needs_setup: true, models: [{ slug: "pickle", name: "Pickle", is_default: true }, { slug: "gpt", name: "GPT", options: [effort] }] },
    ];
    defaults = { provider: "claude", model: "claude-big" };
    mocks.post.mockImplementation(async () => {
      setup = {
        ...emptySetup,
        turns: [
          { ...turn("claudeagent", "Claude", "confirmed", "Confirmed with 12 skills", "r1", "t2"), model: "claude-big" },
          turn("opencode", "OpenCode", "failed", "npx: command not found", "r1", "t3"),
        ],
      };
      return { data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "claudeagent", name: "Claude", model: "claude-big" }, { provider: "opencode", name: "OpenCode" }] } };
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
    await user.click(screen.getByRole("button", { name: /start setup/i }));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", {
        models: { claudeagent: "claude-big", opencode: "" },
        model_options: { claudeagent: [{ id: "effort", value: "high" }], opencode: [] },
        folder: "",
        providers: ["claudeagent", "opencode"],
      }),
    );
    await user.click(await screen.findByRole("button", { name: /^claude: confirmed/i }));
    expect(await screen.findByText("claude-big")).toBeInTheDocument();

    await pickOption(user, "OpenCode model", "GPT");
    await user.click(await screen.findByRole("button", { name: /^retry$/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/opencode/retry", { model: "gpt", model_options: [], folder: "" }));
  });

  describe("choosing which providers to set up", () => {
    beforeEach(() => {
      providers = [
        { id: "codex", driver: "codex", name: "Codex", needs_setup: true, models: [{ slug: "gpt", name: "GPT", is_default: true }] },
        { id: "claude", driver: "claudeAgent", name: "Claude", needs_setup: true, models: [{ slug: "claude-small", name: "Small", is_default: true }] },
        { id: "opencode", driver: "opencode", name: "OpenCode", needs_setup: true, models: [{ slug: "pickle", name: "Pickle", is_default: true }] },
      ];
    });

    it("runs only the included providers and counts confirmations over them alone", async () => {
      setup = { ...emptySetup, turns: [turn("codex", "Codex", "confirmed", "Confirmed with 12 skills", "r0", "t0")] };
      mocks.post.mockImplementation(async () => {
        setup = { ...setup, turns: [...setup.turns, turn("claudeagent", "Claude", "running", "Connecting Nexul and installing skills", "r1", "t1")] };
        return { data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "claudeagent", name: "Claude" }, { provider: "opencode", name: "OpenCode" }] } };
      });
      const user = userEvent.setup();
      renderDialog(paired);

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await user.click(await screen.findByRole("switch", { name: "Codex" }));
      expect(screen.getByRole("combobox", { name: "Codex model" })).toBeDisabled();
      await user.click(screen.getByRole("button", { name: /re-run setup/i }));

      await waitFor(() =>
        expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", {
          models: { codex: "gpt", claudeagent: "claude-small", opencode: "pickle" },
          model_options: { codex: [], claudeagent: [], opencode: [] },
          folder: "",
          providers: ["claudeagent", "opencode"],
        }),
      );
      const list = within(await screen.findByRole("list", { name: "Providers" }));
      expect(list.queryByRole("button", { name: /codex/i })).not.toBeInTheDocument();
      expect(screen.getByText("0/2 confirmed")).toBeInTheDocument();
    });

    it("opens with what the last run skipped switched off, and needs at least one provider on to start", async () => {
      setup = { ...emptySetup, skipped_providers: ["codex"] };
      const user = userEvent.setup();
      renderDialog(paired);

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("switch", { name: "Codex" })).not.toBeChecked();
      expect(screen.getByRole("switch", { name: "Claude" })).toBeChecked();
      expect(screen.getByRole("button", { name: /start setup/i })).toBeEnabled();

      await user.click(screen.getByRole("switch", { name: "Claude" }));
      await user.click(screen.getByRole("switch", { name: "OpenCode" }));
      expect(screen.getByRole("button", { name: /start setup/i })).toBeDisabled();
      expect(screen.getByText(/turn on at least one provider/i)).toBeInTheDocument();
      expect(mocks.post).not.toHaveBeenCalled();
    });
  });

  it("runs setup in the default project's folder, or the one picked, and retries in it too", async () => {
    projects = [
      { id: "t3-gone", title: "StreamerBotChat", path: "/home/me/StreamerBotChat" },
      { id: "t3-app", title: "App", path: "/home/me/app" },
    ];
    defaults = { default_computer_id: "c1", fallback_project_id: "t3-gone" };
    mocks.post.mockImplementation(async () => {
      setup = { ...emptySetup, turns: [turn("codex", "Codex", "failed", "workspace folder no longer exists", "r1", "t2")] };
      return { data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "codex", name: "Codex" }] } };
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^open$/i }));
    expect(await screen.findByRole("combobox", { name: "Folder" })).toHaveTextContent("StreamerBotChat");
    await user.click(screen.getByRole("button", { name: /start setup/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs", { models: {}, model_options: {}, folder: "/home/me/StreamerBotChat" }));

    await pickOption(user, "Folder", /^App/);
    expect(screen.getByText("/home/me/app")).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /^retry$/i }));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/codex/retry", { model: "", model_options: [], folder: "/home/me/app" }),
    );
  });

  describe("closing", () => {
    beforeEach(() => {
      providers = [
        { id: "codex", driver: "codex", name: "Codex", needs_setup: true, models: [{ slug: "gpt", name: "GPT", is_default: true }] },
        { id: "claude", driver: "claudeAgent", name: "Claude", needs_setup: true, models: [{ slug: "claude-big", name: "Big", options: [effort] }, { slug: "claude-small", name: "Small", is_default: true }] },
      ];
      projects = [{ id: "t3-app", title: "App", path: "/home/me/app" }];
    });

    it("stays open on a click outside it", async () => {
      // The modal sets pointer-events none on the page behind it; the click still has to reach the outside handler.
      const user = userEvent.setup({ pointerEventsCheck: 0 });
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await screen.findByRole("switch", { name: "Codex" });

      await user.click(document.body);
      expect(screen.getByRole("heading", { name: /set up work laptop/i })).toBeInTheDocument();
    });

    it("Cancel drops an unsaved change, so the next open shows what was saved", async () => {
      const user = userEvent.setup();
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await user.click(await screen.findByRole("switch", { name: "Codex" }));
      expect(screen.getByRole("switch", { name: "Codex" })).not.toBeChecked();

      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(screen.queryByRole("heading", { name: /set up work laptop/i })).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("switch", { name: "Codex" })).toBeChecked();
      expect(mocks.put).not.toHaveBeenCalled();
    });

    it("Done saves the switches, models, options, and folder, and the next open shows them", async () => {
      mocks.put.mockImplementation(async (_url: string, body: Omit<ComputerSetup, "computer_id" | "confirmed_at" | "providers" | "turns">) => {
        setup = { ...setup, ...body };
        return { data: setup };
      });
      const user = userEvent.setup();
      renderDialog(paired);
      await user.click(screen.getByRole("button", { name: /^open$/i }));
      await user.click(await screen.findByRole("switch", { name: "Codex" }));
      await pickOption(user, "Claude model", "Big");
      await user.click(screen.getByRole("button", { name: "Claude model options" }));
      await user.click(await screen.findByRole("menuitemradio", { name: "High" }));

      await user.click(screen.getByRole("button", { name: "Done" }));
      await waitFor(() =>
        expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/choices", {
          skipped_providers: ["codex"],
          models: { codex: "gpt", claudeagent: "claude-big" },
          model_options: { codex: [], claudeagent: [{ id: "effort", value: "high" }] },
          folder: "/home/me/app",
        }),
      );
      await waitFor(() => expect(screen.queryByRole("heading", { name: /set up work laptop/i })).not.toBeInTheDocument());

      await user.click(screen.getByRole("button", { name: /^open$/i }));
      expect(await screen.findByRole("switch", { name: "Codex" })).not.toBeChecked();
      expect(screen.getByRole("combobox", { name: "Claude model" })).toHaveTextContent("Big");
      expect(screen.getByRole("button", { name: "Claude model options" })).toHaveTextContent("High");
      expect(mocks.post).not.toHaveBeenCalled();
    });
  });
});
