import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { renderPlaysPage } from "@/test/renderPlaysPage";
import { expectWriteControls } from "@/test/gates";
import { asOwner, asReader, settle } from "@/test/viewer";

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
  builtin_key: "",
  created_by: "",
  created_at: "",
  updated_at: "",
  can_run: true,
  ...overrides,
});

const rest = (plays: unknown[]) => (url: string) => {
  if (url === "/api/workspaces/ws-1/plays") return plays;
  if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
  if (url === "/api/projects") return [{ id: "p-1", name: "Website" }];
  if (url === "/api/workspaces/ws-1/plays/pl-x/auto-plays") return [];
  if (url === "/api/workspaces/ws-1/plays/auto-play-limits") return { daily_cap_per_ticket: 5 };
  if (url === "/api/plays/runs") return { items: [], total: 0 };
  throw new Error(`unexpected GET ${url}`);
};

const serve = (plays: unknown[]) => mocks.get.mockImplementation(asOwner(rest(plays)));

beforeEach(() => {
  for (const fn of Object.values(mocks)) fn.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useWorkspaceStore.persist.clearStorage();
});

describe("the plays list", () => {
  it("shows an error when the plays fail to load", async () => {
    mocks.errorMessage.mockReturnValue("Plays failed");
    mocks.get.mockImplementation(
      asOwner((url) => {
        if (url === "/api/workspaces") return [];
        throw new Error("boom");
      }),
    );
    renderPlaysPage();
    expect(await screen.findByText("Plays failed")).toBeInTheDocument();
  });

  it("says there are no plays in the list and in the pane beside it", async () => {
    serve([]);
    renderPlaysPage();
    expect(await screen.findAllByText("No plays yet")).toHaveLength(2);
    expect(screen.getByText("Select a play")).toBeInTheDocument();
  });

  it("groups plays by type in the order Ticket, Doc, Stack, Interview and says what each shows on and whether it is off", async () => {
    serve([
      play({ id: "i", label: "Interview a project", type: "interview", show_when_stage: null }),
      play({ id: "s", label: "Investigate a crash", type: "stack", show_when_stage: null, enabled: false }),
      play({ id: "t", label: "Fix with AI" }),
    ]);
    renderPlaysPage();
    await screen.findByText("3 plays in Acme");

    expect(screen.getAllByRole("region").map((group) => group.getAttribute("aria-label"))).toEqual(["Ticket", "Stack", "Interview"]);
    expect(within(screen.getByRole("region", { name: "Ticket" })).getByText("On In progress")).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "Stack" })).getByText("Off")).toBeInTheDocument();
  });

  it("says how many auto plays start a play by itself, between its stage and Off, and nothing when none is on", async () => {
    serve([
      play({ id: "on", label: "Fix with AI", auto_plays_on: 2, enabled: false }),
      play({ id: "off", label: "Test with AI", type: "doc", show_when_stage: null, auto_plays_on: 0 }),
      play({ id: "unknown", label: "Clarify", type: "doc", show_when_stage: null }),
    ]);
    renderPlaysPage();
    await screen.findByText("3 plays in Acme");

    expect(screen.getAllByText(/ auto on$/).map((mark) => mark.parentElement?.textContent)).toEqual(["On In progress·2 auto on·Off"]);
  });

  it.each([
    { name: "hides", showLocked: false },
    { name: "locks", showLocked: true },
  ])("$name New play for a reader", async ({ showLocked }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["plays:read"] }, rest([play()])));
    renderPlaysPage();
    await screen.findByText("Fix with AI");
    await expectWriteControls([{ role: "button", name: "New play", reason: "Your role can't create plays" }], showLocked);
  });

  it("creates a play and opens it", async () => {
    const user = userEvent.setup();
    serve([]);
    mocks.post.mockResolvedValue({ data: play({ id: "pl-x", label: "Triage" }) });
    const router = renderPlaysPage();

    await user.click(await screen.findByRole("button", { name: "New play" }));
    await user.type(await screen.findByLabelText("Label"), "Triage");
    await user.click(screen.getByRole("button", { name: "Create play" }));

    await waitFor(() => expect(router.state.location.pathname).toBe("/acme/plays/pl-x"));
    expect(mocks.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays", expect.objectContaining({ label: "Triage", type: "ticket", show_when_stage: "progress" }));
  });
});

describe("an open play", () => {
  it("opens from its row and reads what the play is", async () => {
    const user = userEvent.setup();
    serve([play()]);
    const router = renderPlaysPage();

    await user.click(await screen.findByRole("link", { name: /Fix with AI/ }));

    expect(router.state.location.pathname).toBe("/acme/plays/play-fix");
    expect(await screen.findByRole("heading", { name: "Fix with AI", level: 1 })).toBeInTheDocument();
    expect(screen.getByText("Ticket play")).toBeInTheDocument();
    expect(screen.getByText("Reads the ticket and opens a PR.")).toBeInTheDocument();
    expect(screen.getByText("Shows on")).toBeInTheDocument();
    expect(await screen.findByText("Website")).toBeInTheDocument();
    expect(screen.getByText(/Read the ticket\./)).toHaveTextContent("Read the ticket. Open a PR.");
  });

  it("turns the play off from the Enabled switch at once, sending its other values as they are", async () => {
    const user = userEvent.setup();
    serve([play()]);
    mocks.patch.mockResolvedValue({ data: play({ enabled: false }) });
    renderPlaysPage("/acme/plays/play-fix");

    await waitFor(() => expect(screen.getByRole("switch", { name: "Enabled" })).toBeEnabled());
    await user.click(screen.getByRole("switch", { name: "Enabled" }));

    await waitFor(() =>
      expect(mocks.patch).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix", {
        label: "Fix with AI",
        type: "ticket",
        description: "Reads the ticket and opens a PR.",
        instructions: "Read the ticket.\nOpen a PR.",
        enabled: false,
        show_when_stage: "progress",
        excluded_project_ids: ["p-1"],
      }),
    );
  });

  it.each([
    { name: "disables", showLocked: false },
    { name: "locks", showLocked: true },
  ])("$name the Enabled switch and Edit for a reader", async ({ showLocked }) => {
    mocks.get.mockImplementation(asReader({ showLocked, permissions: ["plays:read", "autoplays:read"] }, rest([play()])));
    renderPlaysPage("/acme/plays/play-fix");
    await screen.findByRole("heading", { name: "Fix with AI", level: 1 });
    await expectWriteControls(
      [
        { role: "switch", name: "Enabled", reason: "Your role can't edit plays" },
        { role: "button", name: "Edit", reason: "Your role can't edit plays" },
      ],
      showLocked,
    );
  });

  it("has no auto plays section on an interview play", async () => {
    mocks.get.mockImplementation(asReader({ permissions: ["plays:read", "autoplays:read"] }, rest([play({ id: "i", label: "Interview a project", type: "interview", show_when_stage: null })])));
    renderPlaysPage("/acme/plays/i");
    await screen.findByRole("heading", { name: "Interview a project", level: 1 });
    await settle();
    expect(screen.queryByRole("region", { name: "Starts by itself" })).not.toBeInTheDocument();
  });

  it("has no auto plays section without autoplays:read", async () => {
    mocks.get.mockImplementation(asReader({ permissions: ["plays:read"] }, rest([play()])));
    renderPlaysPage("/acme/plays/play-fix");
    await screen.findByRole("heading", { name: "Fix with AI", level: 1 });
    await settle();
    expect(screen.queryByRole("region", { name: "Starts by itself" })).not.toBeInTheDocument();
  });

  it("deletes the open play after confirming and goes back to the list", async () => {
    const user = userEvent.setup();
    serve([play()]);
    mocks.delete.mockResolvedValue({});
    const router = renderPlaysPage("/acme/plays/play-fix");

    await user.click(await screen.findByRole("button", { name: "More actions for Fix with AI" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
    await user.click(await screen.findByRole("button", { name: "Delete play" }));

    expect(mocks.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-fix");
    await waitFor(() => expect(router.state.location.pathname).toBe("/acme/plays"));
  });

  it("says so for a play that isn't in the list", async () => {
    serve([play()]);
    renderPlaysPage("/acme/plays/gone");
    expect(await screen.findByText("That play isn't here")).toBeInTheDocument();
  });
});
