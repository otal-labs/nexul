import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AutoPlay } from "@/models/AutoPlay";
import { pickOption } from "@/test/pickOption";

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
  created_by: "u1",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const WRITE = ["plays:read", "plays:write", "autoplays:read", "autoplays:write", "autoplays:delete"];
let permissions: string[] = WRITE;
let autoPlays: AutoPlay[] = [];

const serve = () =>
  mocks.get.mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    const projectId = (config?.params as { project_id?: string } | undefined)?.project_id;
    if (url === "/api/workspaces/ws-1/plays") return { data: [play] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/workspaces/ws-1/plays/pl-1/auto-plays") return { data: autoPlays };
    if (url === "/api/workspaces/ws-1/plays/auto-play-limits") return { data: { daily_cap_per_ticket: 5 } };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Website" }] };
    if (url === "/api/ticket-types" && projectId === "p-1") return { data: [{ id: "tt-1", name: "Bug" }] };
    if (url === "/api/tickets/labels") return { data: ["urgent"] };
    if (url === "/api/categories") return { data: [] };
    if (url.startsWith("/api/permissions")) return { data: { grants: [] } };
    throw new Error(`unexpected GET ${url}`);
  });

const openAutoPlays = async () => {
  const user = userEvent.setup();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <PlaySettingsSection canWrite canDelete />
    </QueryClientProvider>,
  );
  await user.click(await screen.findByRole("button", { name: "Actions for Fix with AI" }));
  await user.click(await screen.findByRole("button", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog", { name: "Edit Fix with AI" });
  await user.click(await within(dialog).findByRole("tab", { name: "Auto plays" }));
  return { user, dialog };
};

const footer = (dialog: HTMLElement, name: string) => within(dialog).getAllByRole("button", { name }).at(-1)!;

// The dialogs live in a module-level root, so each test closes its own from the list or the next one opens behind it.
const close = async (user: ReturnType<typeof userEvent.setup>, dialog: HTMLElement) => {
  await within(dialog).findAllByRole("button", { name: /becomes unblocked|Add auto play/ });
  await user.click(footer(dialog, "Cancel"));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
};

beforeEach(() => {
  permissions = WRITE;
  autoPlays = [autoPlay()];
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  serve();
});

describe("a play's auto plays", () => {
  it("lists each as a sentence, switches it at once and puts it back when the save is refused, and sets the daily cap", async () => {
    const { user, dialog } = await openAutoPlays();
    const row = await within(dialog).findByRole("button", { name: /becomes unblocked/ });
    await waitFor(() =>
      expect(row).toHaveTextContent("When a ticket becomes unblocked, if Type is Bug → High, else Normal, runs on Developer"),
    );

    let refuse: (error: Error) => void = () => {};
    mocks.patch.mockImplementation(() => new Promise((_, reject) => (refuse = reject)));
    const toggle = within(dialog).getByRole("switch", { name: "Switch auto play ap-1 on or off" });
    await user.click(toggle);
    expect(toggle).not.toBeChecked();
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1",
      expect.objectContaining({ enabled: false, moment: "ticket.unblocked" }),
    );
    refuse(new Error("no"));
    await waitFor(() => expect(toggle).toBeChecked());

    mocks.patch.mockResolvedValue({ data: { daily_cap_per_ticket: 7 } });
    await pickOption(user, "Auto plays per ticket a day", "7");
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/workspaces/ws-1/plays/auto-play-limits", { daily_cap_per_ticket: 7 });
    await close(user, dialog);
  });

  it("opens one in place of the list, and Cancel with an edit asks before going back", async () => {
    const { user, dialog } = await openAutoPlays();
    await user.click(await within(dialog).findByRole("button", { name: /becomes unblocked/ }));
    await pickOption(user, "Moment", "Ticket is created");

    await user.click(footer(dialog, "Cancel"));
    const prompt = await screen.findByRole("dialog", { name: "Discard changes?" });
    await user.click(within(prompt).getByRole("button", { name: "Cancel" }));
    expect(within(dialog).getByRole("combobox", { name: "Moment" })).toHaveTextContent("Ticket is created");

    await user.click(footer(dialog, "Cancel"));
    await user.click(within(await screen.findByRole("dialog", { name: "Discard changes?" })).getByRole("button", { name: "Discard changes" }));
    expect(await within(dialog).findByRole("button", { name: /becomes unblocked/ })).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "Edit Fix with AI" })).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
    await close(user, dialog);
  });

  it("builds a condition and a nested group, and Save creates the auto play and goes back to the list", async () => {
    autoPlays = [];
    const { user, dialog } = await openAutoPlays();
    expect(await within(dialog).findByText(/No auto plays yet/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Add auto play" }));

    await user.click(within(dialog).getByRole("button", { name: "Add a condition" }));
    await user.click(within(dialog).getByRole("button", { name: "Type values" }));
    await user.click(await screen.findByRole("checkbox", { name: "Bug" }));
    await user.keyboard("{Escape}");

    await user.click(within(dialog).getByRole("button", { name: "Add a group" }));
    const fields = within(dialog).getAllByRole("combobox", { name: "Field" });
    await user.click(fields[1]!);
    await user.click(await screen.findByRole("option", { name: "Label" }));
    await user.click(within(dialog).getByRole("button", { name: "Label values" }));
    await user.click(await screen.findByRole("checkbox", { name: "urgent" }));
    await user.keyboard("{Escape}");
    expect(within(dialog).getAllByText(/must match:/)[0]).toHaveTextContent("of these 2 conditions must match:");

    mocks.post.mockResolvedValue({ data: autoPlay({ id: "ap-2", enabled: false }) });
    await user.click(footer(dialog, "Save"));
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
      }),
    );
    expect(await within(dialog).findByRole("button", { name: "Add auto play" })).toBeInTheDocument();
    await close(user, dialog);
  });

  it("shows the list and the composer read-only without autoplays:write", async () => {
    permissions = ["plays:read", "plays:write", "autoplays:read"];
    const { user, dialog } = await openAutoPlays();
    const toggle = await within(dialog).findByRole("switch", { name: "Switch auto play ap-1 on or off" });
    expect(toggle).toBeDisabled();
    expect(within(dialog).queryByRole("button", { name: "Add auto play" })).not.toBeInTheDocument();
    expect(await within(dialog).findByText(/auto plays a day/)).toHaveTextContent("Each ticket runs at most 5 auto plays a day");

    await user.click(within(dialog).getByRole("button", { name: /becomes unblocked/ }));
    expect(within(dialog).getByRole("combobox", { name: "Moment" })).toBeDisabled();
    expect(within(dialog).queryByRole("button", { name: "Add a condition" })).not.toBeInTheDocument();
    await user.click(footer(dialog, "Save"));
    expect(await within(dialog).findByRole("button", { name: /becomes unblocked/ })).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
    await close(user, dialog);
  });

  it("keeps a value it can't name through a save, shown as an unknown project", async () => {
    autoPlays = [
      autoPlay({ conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "project", op: "is", values: ["p-gone", "p-1"] }] }] } }),
    ];
    const { user, dialog } = await openAutoPlays();
    await user.click(await within(dialog).findByRole("button", { name: /becomes unblocked/ }));
    await waitFor(() =>
      expect(within(dialog).getByRole("button", { name: "Project values" })).toHaveTextContent("Unknown project, Website"),
    );

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer(dialog, "Save"));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith(
        "/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1",
        expect.objectContaining({
          conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "project", op: "is", values: ["p-gone", "p-1"] }] }] },
        }),
      ),
    );
    await close(user, dialog);
  });
});
