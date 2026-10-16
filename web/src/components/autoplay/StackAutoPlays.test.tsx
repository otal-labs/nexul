import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";
import { pickOption } from "@/test/pickOption";

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
  conditions: { match: "all", groups: [] },
  priority: { rules: [], otherwise: "normal" },
  once_within_minutes: 0,
  run_on: "computer",
  computer_id: "",
  run_as: "",
  created_by: "",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const computers = [
  { id: "c-1", name: "Alice laptop" },
  { id: "c-2", name: "Build box", shared: { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see", "computers:run_agents"] } },
  { id: "c-3", name: "Phone", shared: { owner_id: "u-carol", owner_name: "Carol", permissions: ["computers:see", "computers:read_logs"] } },
];

const WRITE = ["plays:read", "plays:write", "autoplays:read", "autoplays:write", "autoplays:delete"];
let current: Play = play();
let autoPlays: AutoPlay[] = [];

const serve = () =>
  mocks.get.mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    const projectId = (config?.params as { project_id?: string } | undefined)?.project_id;
    if (url === "/api/workspaces/ws-1/plays") return { data: [current] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: WRITE } };
    if (url === "/api/workspaces/ws-1/plays/pl-1/auto-plays") return { data: autoPlays };
    if (url === "/api/workspaces/ws-1/plays/auto-play-limits") return { data: { daily_cap_per_ticket: 5 } };
    if (url === "/api/workspaces/ws-1/plays/pl-1/queued") return { data: [] };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Website" }] };
    if (url === "/api/stacks") return { data: [{ id: "st-1", name: "acme-web", project_id: projectId ?? "p-1" }] };
    if (url === "/api/stacks/st-1/services") return { data: [{ id: "sv-1", stack_id: "st-1", name: "api" }, { id: "sv-2", stack_id: "st-1", name: "worker" }] };
    if (url === "/api/pairing/computers") return { data: { computers } };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url.startsWith("/api/permissions")) return { data: { grants: [] } };
    return { data: [] };
  });

const openAutoPlays = async (label: string) => {
  const user = userEvent.setup();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <PlaySettingsSection />
    </QueryClientProvider>,
  );
  await user.pointer({ keys: "[MouseRight]", target: await screen.findByText(label) });
  await user.click(await screen.findByRole("menuitem", { name: "Edit" }));
  const dialog = await screen.findByRole("dialog", { name: `Edit ${label}` });
  await user.click(await within(dialog).findByRole("tab", { name: "Auto plays" }));
  return { user, dialog };
};

const footer = (dialog: HTMLElement, name: string) => within(dialog).getAllByRole("button", { name }).at(-1)!;

// Cancel in the composer goes back to the list, and Cancel in the list closes the dialog; edits ask before they are dropped.
const close = async (user: ReturnType<typeof userEvent.setup>, dialog: HTMLElement) => {
  await user.click(footer(dialog, "Cancel"));
  const discard = screen.queryByRole("button", { name: "Discard changes" });
  if (discard) await user.click(discard);
  if (within(dialog).queryByRole("combobox", { name: "Moment" })) await waitFor(() => expect(within(dialog).queryByRole("combobox", { name: "Moment" })).not.toBeInTheDocument());
  if (screen.queryByRole("dialog")) await user.click(footer(dialog, "Cancel"));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
};

beforeEach(() => {
  current = play();
  autoPlays = [];
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  serve();
});

describe("a stack play's auto plays", () => {
  it("reads a row as a sentence about a stack, and says which computer it runs on or that nobody picked one", async () => {
    autoPlays = [autoPlay(), autoPlay({ id: "ap-2", moment: "stack.service_recovered", computer_id: "c-2", run_as: "u-me", enabled: true })];
    const { user, dialog } = await openAutoPlays("Investigate a crash");

    const rows = await within(dialog).findAllByRole("button", { name: /When a stack/ });
    await waitFor(() => expect(rows[0]).toHaveTextContent("When a stack service crashes → Normal, runs on a computer nobody has picked yet"));
    await waitFor(() => expect(rows[1]).toHaveTextContent("When a stack service recovers → Normal, runs on Bob's Build box (shared)"));
    expect(within(dialog).queryByText(/auto plays a day/)).not.toBeInTheDocument();
    await close(user, dialog);
  });

  it("offers the stack moments and conditions, runs on a chosen computer, and saves the pick", async () => {
    const { user, dialog } = await openAutoPlays("Investigate a crash");
    await user.click(await within(dialog).findByRole("button", { name: "Add auto play" }));

    expect(within(dialog).getByRole("combobox", { name: "Moment" })).toHaveTextContent("Service crashes");
    await user.click(within(dialog).getByRole("combobox", { name: "Moment" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Service crashes", "Service recovers"]);
    await user.keyboard("{Escape}");

    const runOn = within(dialog).getByRole("region", { name: "Run on" });
    expect(runOn).toHaveTextContent("A chosen computer");
    expect(within(runOn).queryByRole("combobox", { name: "Run on" })).not.toBeInTheDocument();
    await user.click(within(runOn).getByRole("combobox", { name: "Computer" }));
    const options = await screen.findAllByRole("option");
    expect(options.map((o) => o.textContent)).toEqual([
      "Alice laptop",
      "Bob's Build box (shared) · can't run here yet",
      "Carol's Phone (shared) · can't run here yet",
    ]);
    expect(options.map((o) => o.getAttribute("aria-disabled"))).toEqual([null, "true", "true"]);
    await user.click(screen.getByRole("option", { name: "Alice laptop" }));

    await user.click(within(dialog).getByRole("button", { name: "Add a condition" }));
    await user.click(within(dialog).getByRole("combobox", { name: "Field" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Project", "Stack", "Service"]);
    await user.click(screen.getByRole("option", { name: "Service" }));
    await user.click(within(dialog).getByRole("button", { name: "Service values" }));
    await user.click(await screen.findByRole("checkbox", { name: "api" }));
    await user.keyboard("{Escape}");

    mocks.post.mockResolvedValue({ data: autoPlay({ id: "ap-2" }) });
    await user.click(footer(dialog, "Save"));
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays", {
        enabled: false,
        moment: "stack.service_crashed",
        moment_stage: null,
        conditions: { match: "all", groups: [{ match: "all", rules: [{ field: "service", op: "is", values: ["api"] }] }] },
        priority: { rules: [], otherwise: "normal" },
        once_within_minutes: 1440,
        run_on: "computer",
        computer_id: "c-1",
      }),
    );
    await close(user, dialog);
  });

  it("keeps a computer picked by someone else that the viewer can't see, so a save never drops it", async () => {
    autoPlays = [autoPlay({ computer_id: "c-gone", run_as: "u-other" })];
    const { user, dialog } = await openAutoPlays("Investigate a crash");
    await user.click(await within(dialog).findByRole("button", { name: /When a stack/ }));

    expect(await within(dialog).findByRole("combobox", { name: "Computer" })).toHaveTextContent("A computer you can't see");

    mocks.patch.mockResolvedValue({ data: autoPlays[0] });
    await user.click(footer(dialog, "Save"));
    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/pl-1/auto-plays/ap-1", expect.objectContaining({ run_on: "computer", computer_id: "c-gone" })),
    );
    await close(user, dialog);
  });

  it("says so when the viewer has no computer to pick", async () => {
    computers.length = 0;
    try {
      const { user, dialog } = await openAutoPlays("Investigate a crash");
      await user.click(await within(dialog).findByRole("button", { name: "Add auto play" }));

      expect(await within(dialog).findByText(/You have no computer to pick yet/)).toBeInTheDocument();
      expect(within(dialog).queryByRole("combobox", { name: "Computer" })).not.toBeInTheDocument();
      await close(user, dialog);
    } finally {
      computers.push(
        { id: "c-1", name: "Alice laptop" },
        { id: "c-2", name: "Build box", shared: { owner_id: "u-bob", owner_name: "Bob", permissions: ["computers:see", "computers:run_agents"] } },
        { id: "c-3", name: "Phone", shared: { owner_id: "u-carol", owner_name: "Carol", permissions: ["computers:see", "computers:read_logs"] } },
      );
    }
  });
});

describe("a chosen computer for the other play types", () => {
  it("is a run-on choice of a ticket play, which then asks for the computer", async () => {
    current = play({ label: "Fix with AI", type: "ticket", show_when_stage: "progress", builtin_key: "" });
    const { user, dialog } = await openAutoPlays("Fix with AI");
    await user.click(await within(dialog).findByRole("button", { name: "Add auto play" }));

    const runOn = within(dialog).getByRole("region", { name: "Run on" });
    expect(within(runOn).queryByRole("combobox", { name: "Computer" })).not.toBeInTheDocument();
    await pickOption(user, "Run on", "A chosen computer");
    expect(await within(runOn).findByRole("combobox", { name: "Computer" })).toBeInTheDocument();
    await close(user, dialog);
  });

  it("is the one other run-on choice of a doc play, beside whoever caused it", async () => {
    current = play({ label: "To tickets via AI", type: "doc", builtin_key: "" });
    const { user, dialog } = await openAutoPlays("To tickets via AI");
    await user.click(await within(dialog).findByRole("button", { name: "Add auto play" }));

    const runOn = within(dialog).getByRole("region", { name: "Run on" });
    expect(within(runOn).getByRole("combobox", { name: "Run on" })).toHaveTextContent("Whoever caused it");
    await user.click(within(runOn).getByRole("combobox", { name: "Run on" }));
    expect((await screen.findAllByRole("option")).map((o) => o.textContent)).toEqual(["Whoever caused it", "A chosen computer"]);
    await user.keyboard("{Escape}");
    await close(user, dialog);
  });
});
