import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { pickOption } from "@/test/pickOption";
import { renderAgentPage } from "@/test/renderAgentPage";
import { expectWriteControls } from "@/test/gates";
import { asReader, asViewer, settle } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn(), errorMessage: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play = (overrides: Record<string, unknown> = {}) => ({
  id: "play-fix",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "Reads the ticket and opens a PR.",
  instructions: "Read the ticket.\nOpen a PR.",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: ["p-1"],
  repository: "app",
  builtin_key: "",
  created_by: "",
  created_at: "",
  updated_at: "",
  can_run: true,
  runs_7d: 15,
  last_run: { state: "succeeded", started_at: "2026-11-26T10:00:00Z" },
  ...overrides,
});

const autoPlay = (overrides: Record<string, unknown> = {}) => ({
  id: "ap-1",
  play_id: "play-fix",
  workspace_id: "ws-1",
  enabled: true,
  moment: "ticket.entered_stage",
  moment_stage: "progress",
  schedule: null,
  one_at_a_time: false,
  resource: "",
  conditions: { match: "all", groups: [] },
  priority: { rules: [], otherwise: "normal" },
  once_within_minutes: 0,
  run_on: "person",
  run_as: "u-viewer",
  memory_ids: [],
  created_by: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const computer = (id: string, name: string, overrides: Record<string, unknown> = {}) => ({
  id,
  name,
  server_url: "",
  runner: { connected: true, last_seen: "", reconnecting: false },
  position: 0,
  runs_at_once: 1,
  running: 0,
  labels: [],
  derived_labels: [],
  ...overrides,
});

const MINI = computer("c-mini", "onik-mini", { derived_labels: ["linux"] });
const DESK = computer("c-desk", "DESKTOP-4DMA8EK", { runner: { connected: false, last_seen: "", reconnecting: false }, derived_labels: ["windows"] });

const OWNER_ME = {
  role_name: "Owner",
  roles: [{ id: "role-owner", name: "Owner", is_owner: true }],
  permissions: ["plays:read", "plays:write", "plays:run", "plays:delete", "autoplays:read", "autoplays:write", "autoplays:delete"],
};
const asOwner = (rest: (url: string) => unknown) => asViewer(OWNER_ME, rest, { instance_permissions: [], instance_owner: true });

interface World {
  plays: unknown[];
  autoPlays?: Record<string, unknown[]>;
  computers?: unknown[];
}

const rest = ({ plays, autoPlays = {}, computers = [] }: World) => (url: string) => {
  if (url === "/api/pairing/computers") return { computers };
  if (url === "/api/pairing/presence") return { computers: {} };
  if (url === "/api/workspaces/ws-1/plays") return plays;
  if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
  if (url === "/api/projects") return [{ id: "p-1", name: "Website" }];
  if (url.endsWith("/rename-impact")) return { count: 0, automations: [] };
  const auto = /\/plays\/([^/]+)\/auto-plays$/.exec(url);
  if (auto) return autoPlays[auto[1] ?? ""] ?? [];
  return [];
};

const serve = (world: World) => mocks.get.mockImplementation(asOwner(rest(world)));

beforeEach(() => {
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useWorkspaceStore.persist.clearStorage();
});

describe("the tabs", () => {
  it("shows an error when the plays fail to load", async () => {
    mocks.errorMessage.mockReturnValue("Plays failed");
    mocks.get.mockImplementation(
      asOwner((url) => {
        if (url === "/api/workspaces") return [];
        throw new Error("boom");
      }),
    );
    renderAgentPage("/acme/agent/ticket");
    expect(await screen.findByText("Plays failed")).toBeInTheDocument();
  });

  it("opens on Ticket plays, counts each type, and lists only the picked tab's plays", async () => {
    const user = userEvent.setup();
    serve({ plays: [play(), play({ id: "d", label: "Clarify", type: "doc", show_when_stage: null }), play({ id: "d2", label: "Update doc", type: "doc", show_when_stage: null })] });
    const router = renderAgentPage("/acme/agent");

    expect(await screen.findByText("Fix with AI")).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/acme/agent/ticket");
    expect(screen.getByRole("tab", { name: "Ticket plays1" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Doc plays2" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Stack plays0" })).toBeInTheDocument();
    expect(screen.queryByText("Clarify")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New ticket play" })).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Doc plays2" }));
    expect(await screen.findByText("Clarify")).toBeInTheDocument();
    expect(screen.queryByText("Fix with AI")).not.toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/acme/agent/doc");
    expect(screen.getByRole("button", { name: "New doc play" })).toBeInTheDocument();
  });

  it("says an empty tab is empty and offers its New button", async () => {
    serve({ plays: [] });
    renderAgentPage("/acme/agent/stack");
    expect(await screen.findByText("No stack plays yet")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "New stack play" })).toHaveLength(2);
  });

  it("creates a play of the tab's type and opens its row", async () => {
    const user = userEvent.setup();
    serve({ plays: [] });
    mocks.post.mockResolvedValue({ data: play({ id: "pl-x", label: "Triage", type: "doc", show_when_stage: null }) });
    const router = renderAgentPage("/acme/agent/doc");

    await user.click((await screen.findAllByRole("button", { name: "New doc play" }))[0]!);
    await user.type(await screen.findByLabelText("Label"), "Triage");
    await user.click(screen.getByRole("button", { name: "Create play" }));

    await waitFor(() => expect(router.state.location.pathname + router.state.location.search).toBe("/acme/agent/doc?open=pl-x"));
    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays", expect.objectContaining({ label: "Triage", type: "doc", show_when_stage: null }));
  });

  it("opens the Computers dialog by setting ?computers=, with how many of the finished computers are online", async () => {
    const user = userEvent.setup();
    serve({ plays: [], computers: [MINI, DESK, computer("c-new", "", { unfinished: true })] });
    const router = renderAgentPage("/acme/agent/ticket");

    const button = await screen.findByRole("button", { name: /Computers.*1\/2/ });
    await user.click(button);
    expect(router.state.location.search).toBe("?computers=");
  });

  it.each([
    ["/acme/agent/plays/d", "/acme/agent/doc?open=d"],
    ["/acme/agent/plays/gone", "/acme/agent/ticket"],
    ["/acme/agent/computers/c-1", "/acme/agent/ticket?computers=c-1"],
    ["/acme/agent/computers/c-1?setup=c-1", "/acme/agent/ticket?computers=c-1&setup=c-1"],
    ["/acme/agent/projects/defaults", "/acme/agent/ticket?computers=defaults"],
    ["/acme/agent/projects/p-1", "/acme/agent/ticket?computers=project%3Ap-1"],
  ])("sends the old link %s to %s", async (from, to) => {
    serve({ plays: [play({ id: "d", label: "Clarify", type: "doc", show_when_stage: null })] });
    const router = renderAgentPage(from);
    await waitFor(() => expect(router.state.location.pathname + router.state.location.search).toBe(to));
  });
});

describe("the tree", () => {
  it("reads a ticket play's stage, description, runs and last run, and saves its switch at once with its other values", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()] });
    mocks.patch.mockResolvedValue({ data: play({ enabled: false }) });
    renderAgentPage("/acme/agent/ticket");

    const row = await screen.findByRole("button", { name: /Fix with AI/ });
    expect(row).toHaveTextContent("Button on tickets in In progress · Reads the ticket and opens a PR.");
    expect(screen.getByText("15")).toBeInTheDocument();

    await waitFor(() => expect(screen.getByRole("switch", { name: "Switch Fix with AI on or off" })).toBeEnabled());
    await user.click(screen.getByRole("switch", { name: "Switch Fix with AI on or off" }));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", {
        label: "Fix with AI",
        type: "ticket",
        description: "Reads the ticket and opens a PR.",
        instructions: "Read the ticket.\nOpen a PR.",
        enabled: false,
        show_when_stage: "progress",
        excluded_project_ids: ["p-1"],
        repository: "app",
        needs: [],
      }),
    );
  });

  it("reads an auto play as a sentence with the computers it would run on, or whose they are when only known at run time", async () => {
    serve({
      plays: [play()],
      computers: [MINI, DESK],
      autoPlays: {
        "play-fix": [
          autoPlay({ computer_rule: { labels_out: ["windows"] } }),
          autoPlay({ id: "ap-2", moment: "ticket.unblocked", moment_stage: null, run_on: "causer", run_as: "", enabled: false }),
        ],
      },
    });
    renderAgentPage("/acme/agent/ticket");

    const mine = await screen.findByRole("button", { name: /When it enters In progress/ });
    await waitFor(() => expect(mine).toHaveTextContent("onik-mini"));
    expect(mine).not.toHaveTextContent("DESKTOP-4DMA8EK");
    expect(await screen.findByRole("button", { name: /When it becomes unblocked/ })).toHaveTextContent("Whoever moved it: their computers");
    expect(screen.getByRole("button", { name: "+ Start it by itself when…" })).toBeInTheDocument();
  });

  it("gives an interview play no auto plays and no way to add one", async () => {
    serve({ plays: [play({ id: "i", label: "Interview a project", type: "interview", show_when_stage: null })] });
    renderAgentPage("/acme/agent/interview");
    await screen.findByText("Interview a project");
    await settle();
    expect(screen.queryByRole("button", { name: "+ Start it by itself when…" })).not.toBeInTheDocument();
  });

  it.each([
    { name: "hides", showLocked: false },
    { name: "locks", showLocked: true },
  ])("$name a reader's way to add an auto play and shows the switches disabled", async ({ showLocked }) => {
    mocks.get.mockImplementation(
      asReader({ showLocked, permissions: ["plays:read", "autoplays:read"] }, rest({ plays: [play()], autoPlays: { "play-fix": [autoPlay()] } })),
    );
    renderAgentPage("/acme/agent/ticket");
    await screen.findByRole("button", { name: /When it enters In progress/ });
    await expectWriteControls(
      [
        { role: "switch", name: "Switch Fix with AI on or off", reason: "Your role can't edit plays" },
        { role: "switch", name: "Switch auto play ap-1 on or off", reason: "Your role can't switch auto plays on or off" },
        { role: "button", name: "+ Start it by itself when…", reason: "Your role can't add auto plays" },
      ],
      showLocked,
    );
  });
});

describe("a row unfolded in place", () => {
  it("opens a play's settings from its row into the URL, saves them and folds back", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()] });
    mocks.patch.mockResolvedValue({ data: play({ label: "Fix it" }) });
    const router = renderAgentPage("/acme/agent/ticket");

    await user.click(await screen.findByRole("button", { name: /Fix with AI/ }));
    expect(router.state.location.search).toBe("?open=play-fix");
    const editor = await screen.findByRole("group", { name: "Edit Fix with AI" });
    expect(within(editor).getByRole("textbox", { name: "Instructions" })).toHaveValue("Read the ticket.\nOpen a PR.");
    expect(within(editor).getByRole("button", { name: "Projects" })).toHaveTextContent("No projects");

    const name = within(editor).getByRole("textbox", { name: "Name" });
    await user.clear(name);
    await user.type(name, "Fix it");
    await user.click(within(editor).getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", expect.objectContaining({ label: "Fix it", excluded_project_ids: ["p-1"] })));
    await waitFor(() => expect(router.state.location.search).toBe(""));
    expect(screen.queryByRole("group", { name: "Edit Fix with AI" })).not.toBeInTheDocument();
  });

  it("lands on a linked row already open, with its History one click away", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()] });
    renderAgentPage("/acme/agent/ticket?open=play-fix");

    const editor = await screen.findByRole("group", { name: "Edit Fix with AI" });
    await user.click(within(editor).getByRole("button", { name: "History" }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
  });

  it("deletes the play from its editor after confirming", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()] });
    mocks.delete.mockResolvedValue({});
    renderAgentPage("/acme/agent/ticket?open=play-fix");

    const editor = await screen.findByRole("group", { name: "Edit Fix with AI" });
    await user.click(within(editor).getByRole("button", { name: "Delete" }));
    await user.click(await screen.findByRole("button", { name: "Delete play" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix");
  });

  it("saves the computers an auto play is limited to as its computer rule, and lists the ones left out struck through", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()], computers: [MINI, DESK], autoPlays: { "play-fix": [autoPlay()] } });
    mocks.patch.mockResolvedValue({ data: autoPlay() });
    renderAgentPage("/acme/agent/ticket?open=ap-1");

    const editor = await screen.findByRole("group", { name: "Edit auto play" });
    const section = within(editor).getByRole("region", { name: "On computers" });
    await pickOption(user, "Computer choice", "Not labelled");
    await user.click(within(section).getByRole("button", { name: "Labels" }));
    await user.click(await screen.findByRole("checkbox", { name: "windows" }));
    await user.keyboard("{Escape}");

    const left = await within(section).findByText("DESKTOP-4DMA8EK");
    expect(left.parentElement).toHaveClass("line-through");
    expect(within(section).getByText("onik-mini")).not.toHaveClass("line-through");

    await user.click(within(editor).getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith(
        "/api/workspaces/ws-1/plays/play-fix/auto-plays/ap-1",
        expect.objectContaining({ computer_rule: { labels_in: [], labels_out: ["windows"], ids: [] } }),
      ),
    );
  });

  it("offers Only these only when the auto play runs as a fixed person, and drops the ids when it does not", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()], computers: [MINI], autoPlays: { "play-fix": [autoPlay({ run_on: "developer", run_as: "", computer_rule: { ids: ["c-mini"] } })] } });
    mocks.patch.mockResolvedValue({ data: autoPlay() });
    renderAgentPage("/acme/agent/ticket?open=ap-1");

    const editor = await screen.findByRole("group", { name: "Edit auto play" });
    await user.click(within(editor).getByRole("combobox", { name: "Computer choice" }));
    expect(await screen.findByRole("option", { name: "Only these" })).toHaveAttribute("aria-disabled", "true");
    await user.keyboard("{Escape}");

    await user.click(within(editor).getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ computer_rule: { labels_in: [], labels_out: [], ids: [] } })));
  });

  it("starts a new auto play from the muted row under a play, runs as you", async () => {
    const user = userEvent.setup();
    serve({ plays: [play()], autoPlays: { "play-fix": [] } });
    mocks.post.mockResolvedValue({ data: autoPlay() });
    const router = renderAgentPage("/acme/agent/ticket");

    await user.click(await screen.findByRole("button", { name: "+ Start it by itself when…" }));
    expect(router.state.location.search).toBe("?open=new%3Aplay-fix");
    const editor = await screen.findByRole("group", { name: "New auto play" });
    await user.click(within(editor).getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix/auto-plays", expect.objectContaining({ run_on: "developer", enabled: false })));
  });
});
