import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import type { Play } from "@/models/Play";
import type { LatestChoices } from "@/models/Trail";
import { pickOption } from "@/test/pickOption";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => {
    const e = error as { response?: { data?: { message?: string } }; message?: string };
    return e?.response?.data?.message ?? e?.message ?? "Something went wrong";
  }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play: Play = {
  id: "play-1",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "Implement the ticket and open a PR",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  builtin_key: "",
  created_by: "u-1",
  created_at: "",
  updated_at: "",
};

const memories = [
  { id: "m-always", title: "Working in this project", when_to_use: "Always", always_included: true, footer: false },
  { id: "m-react", title: "React guide", when_to_use: "Any change under web/", always_included: false, footer: false },
  { id: "m-go", title: "Go practices", when_to_use: "Any change under internal/", always_included: false, footer: false },
  { id: "m-conclude", title: "Where tickets go next", when_to_use: "Concluding a run", always_included: false, footer: true },
];

const computers = [
  { id: "c-1", name: "Onik's PC" },
  { id: "c-2", name: "VPS" },
];
const providers = [{ id: "claude", driver: "claudeAgent", name: "Claude", models: [{ slug: "sonnet-5", name: "Sonnet 5" }, { slug: "haiku", name: "Haiku" }] }];

let choices: LatestChoices;
let resolve: { ok: boolean; computer_id?: string; provider?: string; model?: string; reason?: string };
let presence: Record<string, string>;

const mockApi = (permissions: string[], memoriesOverride: unknown[] = memories) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/memories") return { data: memoriesOverride };
    if (url === "/api/plays/latest-choices") return { data: choices };
    if (url === "/api/pairing/resolve") return { data: resolve };
    if (url === "/api/pairing/presence") return { data: { computers: presence } };
    if (url === "/api/pairing/computers") return { data: { computers } };
    if (url === "/api/pairing/computers/c-1/providers") return { data: { providers } };
    if (url === "/api/pairing/computers/c-2/providers") return { data: { providers: [] } };
    return { data: [] };
  });

const renderDialog = (open = true) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onClose = vi.fn();
  const ui = (isOpen: boolean) => (
    <QueryClientProvider client={client}>
      <PlayRunDialog play={play} projectId="p-1" targetType="ticket" targetId="t-1" open={isOpen} onClose={onClose} />
    </QueryClientProvider>
  );
  const utils = render(ui(open));
  return { ...utils, onClose, reopen: () => utils.rerender(ui(true)), close: () => utils.rerender(ui(false)) };
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  choices = { memory_ids: ["m-always", "m-react"], computer_id: "", provider: "", model: "" };
  resolve = { ok: true, computer_id: "c-1", provider: "claude", model: "sonnet-5" };
  presence = { "c-1": "connected", "c-2": "connecting" };
});

describe("PlayRunDialog", () => {
  it("pre-selects the remembered memories, locking the always-included memory, and lists footer memories apart", async () => {
    mockApi(["plays:run", "tickets:write"]);
    renderDialog();

    const always = await screen.findByRole("checkbox", { name: "Working in this project" });
    expect(always).toBeChecked();
    expect(always).toBeDisabled();
    expect(screen.getByLabelText("Required")).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "React guide" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Go practices" })).not.toBeChecked();
    expect(screen.getByText("Any change under web/")).toBeInTheDocument();
    const memoriesSection = screen.getByRole("heading", { name: "Main" }).closest("section")!;
    const footerSection = screen.getByRole("heading", { name: "Footer" }).closest("section")!;
    expect(within(memoriesSection).queryByRole("checkbox", { name: "Where tickets go next" })).not.toBeInTheDocument();
    expect(within(footerSection).getByRole("checkbox", { name: "Where tickets go next" })).not.toBeChecked();
    expect(within(footerSection).queryByRole("checkbox", { name: "React guide" })).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "On success, move to" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Run Fix with AI" })).toBeInTheDocument();
  });

  it("normalises a description ending in a period instead of doubling the full stop", async () => {
    mockApi(["plays:run", "tickets:write"]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <PlayRunDialog
          play={{ ...play, description: "Reads the ticket and opens a pull request." }}
          projectId="p-1"
          targetType="ticket"
          targetId="t-1"
          open
          onClose={vi.fn()}
        />
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Reads the ticket and opens a pull request. Runs on your paired harness as you.")).toBeInTheDocument();
  });

  it("posts the chosen memories, footer memories, and instructions, then pre-selects them on the next open", async () => {
    const user = userEvent.setup();
    mockApi(["plays:run", "tickets:write"]);
    vi.mocked(api.post).mockImplementation(async (_url: string, body: unknown) => {
      const input = body as { memory_ids: string[]; computer_id: string; provider: string; model: string };
      choices = {
        memory_ids: ["m-always", ...input.memory_ids],
        computer_id: input.computer_id,
        provider: input.provider,
        model: input.model,
      };
      return { data: { id: "tr-2", play_id: "play-1", play_label: "Fix with AI", target_type: "ticket", target_id: "t-1", project_id: "p-1", state: "starting" } };
    });
    const { onClose, close, reopen } = renderDialog();

    await user.click(await screen.findByRole("checkbox", { name: "React guide" }));
    await user.click(screen.getByRole("checkbox", { name: "Go practices" }));
    await user.type(screen.getByLabelText("Instructions for this run"), "React-only fix");
    await user.click(screen.getByRole("checkbox", { name: "Where tickets go next" }));
    await user.click(screen.getByRole("button", { name: "Run Fix with AI" }));

    expect(api.post).toHaveBeenCalledWith("/api/plays/play-1/run", {
      target_type: "ticket",
      target_id: "t-1",
      memory_ids: ["m-go", "m-conclude"],
      custom_instructions: "React-only fix",
      computer_id: "c-1",
      provider: "claude",
      model: "sonnet-5",
      model_options: [],
    });
    await waitFor(() => expect(onClose).toHaveBeenCalled());

    close();
    reopen();
    expect(await screen.findByRole("checkbox", { name: "Go practices" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "React guide" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "Where tickets go next" })).toBeChecked();
  });

 it("says so when the project has no footer memories", async () => {
    mockApi(["plays:run"], memories.filter((m) => !m.footer));
    renderDialog();
    expect(await screen.findByText("No footer memories. Move a memory to the Footer folder to conclude runs with it.")).toBeInTheDocument();
  });

  it("shows a refusal inside the dialog and keeps it open", async () => {
    const user = userEvent.setup();
    mockApi(["plays:run", "tickets:write"]);
    vi.mocked(api.post).mockRejectedValue({
      response: { status: 400, data: { message: "memory m-9 is not in this project", code: "invalid" } },
    });
    const { onClose } = renderDialog();

    await user.click(await screen.findByRole("button", { name: "Run Fix with AI" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("memory m-9 is not in this project");
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("offers the refused computer's setup when the run is refused for setup", async () => {
    const user = userEvent.setup();
    mockApi(["plays:run", "tickets:write"]);
    vi.mocked(api.post).mockRejectedValue({
      response: {
        status: 400,
        data: {
          message: "@Agent can't use Claude on Onik's PC until its setup is done — run setup for Onik's PC in Settings → T3 Code Setup.",
          code: "INVALID",
          details: { reason: "setup_required", computer_id: "c-1", computer: "Onik's PC", provider_id: "claude", provider: "Claude" },
        },
      },
    });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <PlayRunDialog play={play} projectId="p-1" targetType="ticket" targetId="t-1" open onClose={vi.fn()} />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Run Fix with AI" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("until its setup is done");
    expect(screen.getByRole("link", { name: "Set up Onik's PC" })).toHaveAttribute("href", "/settings/pairing?setup=c-1");
  });

  it("offers no setup link for a refusal that names no computer to set up", async () => {
    const user = userEvent.setup();
    mockApi(["plays:run", "tickets:write"]);
    vi.mocked(api.post).mockRejectedValue({
      response: { status: 400, data: { message: "pairing not configured: unpaired", code: "INVALID", details: { reason: "unpaired" } } },
    });
    renderDialog();

    await user.click(await screen.findByRole("button", { name: "Run Fix with AI" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("pairing not configured");
    expect(screen.queryByRole("link", { name: /Set up/ })).not.toBeInTheDocument();
  });

  it("preselects the resolved harness when the caller never ran this play here before", async () => {
    mockApi(["plays:run", "tickets:write"]);
    renderDialog();
    expect(await screen.findByRole("button", { name: "Onik's PC · Claude · Sonnet 5" })).toBeInTheDocument();
  });

  it("preselects the caller's last harness choice over the resolved target", async () => {
    choices.computer_id = "c-2";
    choices.provider = "";
    choices.model = "";
    mockApi(["plays:run", "tickets:write"]);
    renderDialog();
    expect(await screen.findByRole("button", { name: "VPS · Provider default · Model default" })).toBeInTheDocument();
  });

  it("changing the harness in the popover is honoured by the run", async () => {
    const user = userEvent.setup();
    presence["c-2"] = "connected";
    mockApi(["plays:run", "tickets:write"]);
    renderDialog();

    await user.click(await screen.findByRole("button", { name: "Onik's PC · Claude · Sonnet 5" }));
    await pickOption(user, "Computer", "VPS");
    await user.click(await screen.findByRole("button", { name: "Run Fix with AI" }));

    expect(api.post).toHaveBeenCalledWith(
      "/api/plays/play-1/run",
      expect.objectContaining({ computer_id: "c-2", provider: "", model: "" }),
    );
  });

  it("cannot pick an offline computer", async () => {
    const user = userEvent.setup();
    mockApi(["plays:run", "tickets:write"]);
    renderDialog();

    await user.click(await screen.findByRole("button", { name: "Onik's PC · Claude · Sonnet 5" }));
    await user.click(await screen.findByRole("combobox", { name: "Computer" }));
    const vps = await screen.findByRole("option", { name: /VPS/ });
    expect(vps).toHaveTextContent("Offline");
    expect(vps).toHaveAttribute("aria-disabled", "true");
  });

  it("asks before running on a blocked ticket, naming what it still waits on", async () => {
    mockApi(["plays:run", "tickets:write"]);
    const base = vi.mocked(api.get).getMockImplementation()!;
    const blocker = { id: "t-2", project_id: "p-1", prefix: "BKS", number: 2, title: "backend", status: "open", done: false };
    const cleared = { ...blocker, id: "t-3", number: 3, done: true };
    vi.mocked(api.get).mockImplementation(async (url: string, config?: unknown) => {
      if (url === "/api/tickets/t-1/ticket-links") {
        return { data: { found_in: null, origin_unknown: false, bugs_found: [], blocks: [], blocked_by: [blocker, cleared], blocked: true } };
      }
      return base(url, config as never);
    });
    vi.mocked(api.post).mockResolvedValue({ data: { id: "tr-9", target_type: "ticket", target_id: "t-1", play_label: "Fix with AI" } });
    const user = userEvent.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <PlayRunDialog play={play} projectId="p-1" targetType="ticket" targetId="t-1" open onClose={vi.fn()} />
        <ContextAwareConfirmation.ConfirmationRoot />
      </QueryClientProvider>,
    );

    const run = await screen.findByRole("button", { name: "Run Fix with AI" });
    await user.click(run);
    expect(await screen.findByText("This ticket is blocked")).toBeInTheDocument();
    expect(screen.getByText("It still waits on BKS-2. Run Fix with AI anyway?")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByText("This ticket is blocked")).not.toBeInTheDocument());
    expect(api.post).not.toHaveBeenCalled();

    await user.click(run);
    await user.click(await screen.findByRole("button", { name: "Run anyway" }));
    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/plays/play-1/run", expect.objectContaining({ target_id: "t-1" })));
  });

  describe("Clarify via AI", () => {
    const clarify: Play = { ...play, id: "play-clarify", label: "Clarify via AI", type: "doc", show_when_stage: null, builtin_key: "clarify" };
    const question = (id: string, answered: Partial<{ selected: string[]; text: string; skipped: boolean }> = {}) => ({
      id, doc_id: "d-1", round: 2, position: 1, question: id, why: "", options: [], multi_select: false,
      selected: [], text: "", skipped: false, ...answered,
    });
    const clarification = (questions: unknown[]) => ({
      rounds: [{ doc_id: "d-1", round: 1, questions: [question("old")] }, { doc_id: "d-1", round: 2, questions }],
      running: false, closed: false, can_close: true,
    });

    const renderClarify = (data: unknown) => {
      mockApi(["plays:run", "docs:write"]);
      const fallback = vi.mocked(api.get).getMockImplementation()!;
      vi.mocked(api.get).mockImplementation(async (...args: Parameters<typeof api.get>) =>
        args[0] === "/api/docs/d-1/clarification" ? { data } : fallback(...args),
      );
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(
        <QueryClientProvider client={client}>
          <PlayRunDialog play={clarify} projectId="p-1" targetType="doc" targetId="d-1" open onClose={vi.fn()} />
        </QueryClientProvider>,
      );
    };

    it("signals the newest round's unanswered questions and still offers the run", async () => {
      renderClarify(clarification([question("a"), question("b"), question("c", { skipped: true }), question("d", { text: "Staff" })]));

      expect(await screen.findByText("2 questions in Round 2 are still unanswered")).toBeInTheDocument();
      expect(await screen.findByRole("button", { name: "Run Clarify via AI" })).toBeEnabled();
    });

    it("says nothing once every question of the newest round is answered or skipped", async () => {
      renderClarify(clarification([question("a", { selected: ["Staff"] }), question("b", { skipped: true })]));

      expect(await screen.findByRole("button", { name: "Run Clarify via AI" })).toBeInTheDocument();
      expect(screen.queryByText(/still unanswered/)).not.toBeInTheDocument();
    });
  });
});
