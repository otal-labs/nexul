import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Memory } from "@/models/Memory";
import { pickOption } from "@/test/pickOption";
import { renderPlaysPage } from "@/test/renderPlaysPage";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: mocks,
  errorMessage: (error: unknown) => (error as Error).message,
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play = {
  id: "pl-1",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  builtin_key: "",
  created_by: "",
  created_at: "",
  updated_at: "",
};

const autoPlay = (overrides: Partial<AutoPlay> = {}): AutoPlay => ({
  id: "ap-1",
  play_id: "pl-1",
  workspace_id: "ws-1",
  enabled: true,
  moment: "ticket.unblocked",
  moment_stage: null,
  conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "type", op: "is", values: ["Bug"] }] }] },
  priority: { rules: [{ level: "high", when: { match: "all", rules: [{ field: "type", op: "is", values: ["Bug"] }] } }], otherwise: "normal" },
  once_within_minutes: 1440,
  run_on: "developer",
  computer_id: "",
  run_as: "",
  memory_ids: [],
  created_by: "u1",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const memory = (id: string, title: string, flags: Partial<Memory> = {}) => ({ id, project_id: "p-1", title, always_included: false, footer: false, ...flags });
const MEMORIES = [memory("m-1", "Branching"), memory("m-2", "Ship it", { footer: true }), memory("m-3", "House rules", { always_included: true })];

const WRITE = ["plays:read", "plays:write", "autoplays:read", "autoplays:write", "autoplays:delete"];
let permissions: string[] = WRITE;
let autoPlays: AutoPlay[] = [];

const serve = () =>
  mocks.get.mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    const projectId = (config?.params as { project_id?: string } | undefined)?.project_id;
    if (url === "/api/workspaces/ws-1/plays") return { data: [play] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/workspaces/ws-1/plays/pl-1/auto-plays") return { data: autoPlays };
    if (url === "/api/workspaces/ws-1/plays/auto-play-limits") return { data: { daily_cap_per_ticket: 5, runs_at_once_per_person: 1 } };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Website" }] };
    if (url === "/api/memories") return { data: MEMORIES };
    if (url === "/api/ticket-types" && projectId === "p-1") return { data: [{ id: "tt-1", name: "Bug" }] };
    if (url === "/api/tickets/labels") return { data: ["urgent"] };
    if (url === "/api/categories") return { data: [] };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url.startsWith("/api/permissions")) return { data: { grants: [] } };
    throw new Error(`unexpected GET ${url}`);
  });

const openAutoPlays = async () => {
  const user = userEvent.setup();
  const router = renderPlaysPage("/acme/plays/pl-1/auto-plays");
  await screen.findByRole("tab", { name: "Auto plays", selected: true });
  return { user, router };
};

const footer = (name: string) => screen.getAllByRole("button", { name }).at(-1)!;

const discardPrompt = () => screen.findByRole("dialog", { name: "Discard changes?" });

beforeEach(() => {
  permissions = WRITE;
  autoPlays = [autoPlay()];
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  serve();
});

describe("a play's auto plays", () => {
  it("lists each as a sentence, switches it at once and puts it back when the save is refused, and sets the daily cap", async () => {
    const when = { match: "any" as const, rules: [{ field: "type" as const, op: "is" as const, values: ["Bug"] }, { field: "label" as const, op: "is" as const, values: ["urgent"] }] };
    autoPlays = [autoPlay(), autoPlay({ id: "ap-2", priority: { rules: [{ level: "high", when }], otherwise: "low" } })];
    const { user } = await openAutoPlays();
    const rows = await screen.findAllByRole("button", { name: /becomes unblocked/ });
    await waitFor(() =>
      expect(rows[0]).toHaveTextContent("When a ticket becomes unblocked, if Type is Bug → High if Type is Bug, else Normal, runs on Developer"),
    );
    expect(rows[1]).toHaveTextContent("→ High if Type is Bug or 1 more, else Low, runs on Developer");

    let refuse: (error: Error) => void = () => {};
    mocks.patch.mockImplementation(() => new Promise((_, reject) => (refuse = reject)));
    const toggle = screen.getByRole("switch", { name: "Switch auto play ap-1 on or off" });
    await user.click(toggle);
    expect(toggle).not.toBeChecked();
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1",
      expect.objectContaining({ enabled: false, moment: "ticket.unblocked" }),
    );
    refuse(new Error("no"));
    await waitFor(() => expect(toggle).toBeChecked());

    mocks.patch.mockResolvedValue({ data: { daily_cap_per_ticket: 7, runs_at_once_per_person: 1 } });
    await pickOption(user, "Auto plays per ticket a day", "7");
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/workspaces/ws-1/plays/auto-play-limits", { daily_cap_per_ticket: 7 });
    expect(screen.queryByRole("combobox", { name: /runs at once/i })).not.toBeInTheDocument();
  });

  it("opens one in place of the list, and Cancel with an edit asks before going back", async () => {
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /becomes unblocked/ }));
    await pickOption(user, "Moment", "Ticket is created");

    await user.click(footer("Cancel"));
    await user.click(within(await discardPrompt()).getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("combobox", { name: "Moment" })).toHaveTextContent("Ticket is created");

    await user.click(footer("Cancel"));
    await user.click(within(await discardPrompt()).getByRole("button", { name: "Discard changes" }));
    expect(await screen.findByRole("button", { name: /becomes unblocked/ })).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("keeps the composer's edits when switching to Play is cancelled", async () => {
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /becomes unblocked/ }));
    await pickOption(user, "Moment", "Ticket is created");

    await user.click(screen.getByRole("tab", { name: "Play" }));
    await user.click(within(await discardPrompt()).getByRole("button", { name: "Cancel" }));

    expect(screen.getByRole("tab", { name: "Auto plays" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("combobox", { name: "Moment" })).toHaveTextContent("Ticket is created");
  });

  it("drops the composer's edits and shows Play when switching is confirmed, and asks nothing when clean", async () => {
    const { user, router } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /becomes unblocked/ }));
    await pickOption(user, "Moment", "Ticket is created");

    await user.click(screen.getByRole("tab", { name: "Play" }));
    await user.click(within(await discardPrompt()).getByRole("button", { name: "Discard changes" }));
    await waitFor(() => expect(screen.getByRole("tab", { name: "Play" })).toHaveAttribute("aria-selected", "true"));
    expect(router.state.location.pathname).toBe("/acme/plays/pl-1");
    expect(screen.queryByRole("combobox", { name: "Moment" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Auto plays" }));
    await user.click(await screen.findByRole("button", { name: /becomes unblocked/ }));
    await user.click(screen.getByRole("tab", { name: "Play" }));
    expect(screen.getByRole("tab", { name: "Play" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("dialog", { name: "Discard changes?" })).not.toBeInTheDocument();
  });

  it("builds a condition and a nested group, and Save creates the auto play and goes back to the list", async () => {
    autoPlays = [];
    const { user } = await openAutoPlays();
    expect(await screen.findByText(/No auto plays yet/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add auto play" }));

    await user.click(screen.getByRole("button", { name: "Add a condition" }));
    await user.click(screen.getByRole("button", { name: "Type values" }));
    await user.click(await screen.findByRole("checkbox", { name: "Bug" }));
    await user.keyboard("{Escape}");

    await user.click(screen.getByRole("button", { name: "Add a group" }));
    const fields = screen.getAllByRole("combobox", { name: "Field" });
    await user.click(fields[1]!);
    await user.click(await screen.findByRole("option", { name: "Label" }));
    await user.click(screen.getByRole("button", { name: "Label values" }));
    await user.click(await screen.findByRole("checkbox", { name: "urgent" }));
    await user.keyboard("{Escape}");
    expect(screen.getAllByText(/must match:/)[0]).toHaveTextContent("of these 2 conditions must match:");

    await user.click(screen.getByRole("button", { name: "Memories" }));
    expect(screen.queryByRole("checkbox", { name: /House rules/ })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("checkbox", { name: "Ship it (footer) · Website" }));
    await user.click(screen.getByRole("checkbox", { name: "Branching · Website" }));
    await user.keyboard("{Escape}");

    mocks.post.mockResolvedValue({ data: autoPlay({ id: "ap-2", enabled: false }) });
    await user.click(footer("Save"));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays", {
        enabled: false,
        moment: "ticket.unblocked",
        moment_stage: null,
        conditions: {
          match: "all",
          groups: [
            { match: "all", rules: [{ field: "type", op: "is", values: ["Bug"] }] },
            { match: "any", rules: [{ field: "label", op: "is", values: ["urgent"] }] },
          ],
        },
        priority: { rules: [], otherwise: "normal" },
        once_within_minutes: 1440,
        run_on: "developer",
        computer_id: "",
        memory_ids: ["m-2", "m-1"],
      }),
    );
    expect(await screen.findByRole("button", { name: "Add auto play" })).toBeInTheDocument();
  });

  it("shows the list and the composer read-only without autoplays:write", async () => {
    permissions = ["plays:read", "plays:write", "autoplays:read"];
    const { user } = await openAutoPlays();
    const toggle = await screen.findByRole("switch", { name: "Switch auto play ap-1 on or off" });
    expect(toggle).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Add auto play" })).not.toBeInTheDocument();
    expect(await screen.findByText(/auto plays a day/)).toHaveTextContent("Each ticket runs at most 5 auto plays a day");

    await user.click(screen.getByRole("button", { name: /becomes unblocked/ }));
    expect(screen.getByRole("combobox", { name: "Moment" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Add a condition" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(await screen.findByRole("button", { name: /becomes unblocked/ })).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("keeps its memories when switched, and one it can't name stays picked as an unknown memory", async () => {
    autoPlays = [autoPlay({ memory_ids: ["m-gone", "m-1"] })];
    const { user } = await openAutoPlays();
    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(await screen.findByRole("switch", { name: "Switch auto play ap-1 on or off" }));
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1",
      expect.objectContaining({ enabled: false, memory_ids: ["m-gone", "m-1"] }),
    );

    await user.click(screen.getByRole("button", { name: /becomes unblocked/ }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Memories" })).toHaveTextContent("Unknown memory, Branching · Website"));
  });

  it("keeps a value it can't name through a save, shown as an unknown project", async () => {
    autoPlays = [
      autoPlay({ conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "project", op: "is", values: ["p-gone", "p-1"] }] }] } }),
    ];
    const { user } = await openAutoPlays();
    await user.click(await screen.findByRole("button", { name: /becomes unblocked/ }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Project values" })).toHaveTextContent("Unknown project, Website"));

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer("Save"));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith(
        "/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1",
        expect.objectContaining({
          conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "project", op: "is", values: ["p-gone", "p-1"] }] }] },
        }),
      ),
    );
  });
});
