import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Computer } from "@/models/Pairing";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { renderComputers } from "@/test/agentComputers";
import { pickOption } from "@/test/pickOption";
import { asOwner } from "@/test/viewer";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), post: vi.fn(), del: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, patch: mocks.patch, post: mocks.post, delete: mocks.del },
  errorMessage: (e: { message?: string }) => e?.message ?? "Something went wrong",
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

const computer = (id: string, name: string, position: number, running = 0): Computer => ({
  id,
  name,
  position,
  runs_at_once: 2,
  running,
  chat_project_id: "",
  skills_outdated: false,
  shell_enabled: true,
  server_url: `http://${id}.nexul-computer.invalid`,
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.1.1",
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
  runner: { connected: true, last_seen: "2026-10-22T12:00:00Z", reconnecting: false },
  facts: { t3: { state: "answering" }, providers: [], projects: [{ id: "t3-notes", title: "notes", path: "/home/alice/notes" }] },
});

let computers: Computer[];

const rest = (url: string) => {
  if (url === "/api/pairing/computers") return { computers };
  if (url === "/api/pairing/presence") return { computers: { "c-laptop": "connected", "c-box": "connected" } };
  if (url === "/api/runners/enrollments") return [];
  if (url === "/api/pairing/defaults") return {};
  if (url === "/api/pairing/projects") return { links: [] };
  if (url === "/api/projects") return [];
  if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
  if (url.endsWith("/setup")) return { computer_id: "", confirmed_at: "2026-10-01T00:00:00Z", providers: [{ provider: "codex", confirmed_at: "2026-10-01T00:00:00Z", skills: [], skills_version: "v1", skills_outdated: false }], turns: [], skipped_providers: [], models: { codex: "gpt-5" }, model_options: {}, folder: "" };
  if (url.endsWith("/providers")) return { providers: [] };
  throw new Error(`unexpected GET ${url}`);
};

const yourComputers = () => screen.getByRole("region", { name: "Your computers" });
// Each row's name in the list, top to bottom, without its status dot or its place number.
const order = () => within(yourComputers()).getAllByRole("link").map((link) => link.querySelector("[title]")?.getAttribute("title"));

describe("Your computers", () => {
  beforeEach(() => {
    for (const fn of Object.values(mocks)) fn.mockReset();
    computers = [computer("c-laptop", "laptop", 0, 1), computer("c-box", "build-box", 1)];
    mocks.get.mockImplementation(asOwner(rest));
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
    useWorkspaceStore.persist.clearStorage();
  });

  it("ranks the computers in their order, and says how busy the open one is", async () => {
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=c-laptop");

    expect(await screen.findByText("#1")).toBeInTheDocument();
    expect(screen.getByText("#2")).toBeInTheDocument();
    expect(order()).toEqual(["laptop", "build-box"]);
    expect(await screen.findByText("1 of 2 running")).toBeInTheDocument();

    await user.click(within(yourComputers()).getByRole("link", { name: /^build-box/ }));
    expect(await screen.findByText("0 of 2 running")).toBeInTheDocument();
  });

  it("moves a computer down from its menu, at once, and sends the new order", async () => {
    const user = userEvent.setup();
    mocks.put.mockImplementation(async () => {
      computers = [{ ...computers[1]!, position: 0 }, { ...computers[0]!, position: 1 }];
      return { data: {} };
    });
    renderComputers("/acme/agent?computers=c-laptop");

    await user.click(await screen.findByRole("button", { name: "Actions for laptop" }));
    await user.click(await screen.findByRole("button", { name: "Move down" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/order", { computer_ids: ["c-box", "c-laptop"] });
    await waitFor(() => expect(order()).toEqual(["build-box", "laptop"]));
  });

  it("puts the order back when the server refuses it", async () => {
    const user = userEvent.setup();
    mocks.put.mockRejectedValue(new Error("the order must name each of your computers once"));
    renderComputers("/acme/agent?computers=c-box");

    await user.click(await screen.findByRole("button", { name: "Actions for build-box" }));
    await user.click(await screen.findByRole("button", { name: "Move up" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalled());
    await waitFor(() => expect(order()).toEqual(["laptop", "build-box"]));
  });

  it("sets how many runs a computer takes at once from its details", async () => {
    const user = userEvent.setup();
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderComputers("/acme/agent?computers=c-laptop");

    await pickOption(user, "Runs at once on laptop", "4");

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { runs_at_once: 4 });
    expect(await screen.findByText("1 of 4 running")).toBeInTheDocument();
  });

  it("adds and removes the labels a play's needs match, beside the ones its facts give it", async () => {
    const user = userEvent.setup();
    computers = [{ ...computers[0]!, labels: ["gpu"], derived_labels: ["docker", "linux"] }, computers[1]!];
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderComputers("/acme/agent?computers=c-laptop");

    expect(await screen.findByText("From its facts")).toBeInTheDocument();
    expect(screen.getByText("docker")).toBeInTheDocument();
    await user.type(screen.getByRole("textbox", { name: "Add a label to laptop" }), "Staging-Keys{Enter}");
    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { labels: ["gpu", "staging-keys"] });

    await user.click(await screen.findByRole("button", { name: "Remove gpu" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/pairing/computers/c-laptop", { labels: ["staging-keys"] });
    expect(screen.queryByRole("button", { name: "Remove docker" }), "what its facts give it can't be removed").not.toBeInTheDocument();
  });

  it("saves each of the three switches inverted or as is, apart from one another", async () => {
    const user = userEvent.setup();
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderComputers("/acme/agent?computers=c-laptop");

    const plays = await screen.findByRole("switch", { name: "Plays you start" });
    const auto = screen.getByRole("switch", { name: "Auto plays" });
    const terminal = screen.getByRole("switch", { name: "Terminal" });
    expect([plays, auto, terminal].map((s) => s.getAttribute("aria-checked"))).toEqual(["true", "true", "true"]);

    await user.click(plays);
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/pairing/computers/c-laptop", { plays_off: true });
    await waitFor(() => expect(screen.getByRole("switch", { name: "Plays you start" })).not.toBeChecked());
    expect(screen.getByRole("switch", { name: "Auto plays" }), "Auto plays keeps its own state").toBeChecked();

    await user.click(screen.getByRole("switch", { name: "Auto plays" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/pairing/computers/c-laptop", { auto_plays_off: true });
    await waitFor(() => expect(screen.getByRole("switch", { name: "Auto plays" })).not.toBeChecked());

    await user.click(screen.getByRole("switch", { name: "Terminal" }));
    expect(mocks.patch).toHaveBeenLastCalledWith("/api/pairing/computers/c-laptop", { shell_enabled: false });
  });

  it("offers Open terminal in the header only while the terminal is on and the computer online", async () => {
    computers = [computers[0]!, { ...computers[1]!, shell_enabled: false }];
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=c-laptop");

    expect(await screen.findByRole("button", { name: "Open terminal" })).toBeInTheDocument();
    await user.click(within(yourComputers()).getByRole("link", { name: /^build-box/ }));
    expect(await screen.findByRole("heading", { name: "build-box", level: 3 })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open terminal" })).not.toBeInTheDocument();
  });

  it("shows each set-up provider as its mark, never its model, and a stale offline computer's warning with Remove", async () => {
    const stale = { ...computers[1]!, runner: { connected: false, last_seen: "2026-01-01T00:00:00Z", reconnecting: false } };
    computers = [computers[0]!, stale];
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=c-laptop");

    const row = within((await screen.findByRole("link", { name: /^laptop/ })).closest("li") as HTMLElement);
    expect(await row.findByTitle("codex · GPT 5")).toBeInTheDocument();
    expect(row.queryByText("GPT-5")).not.toBeInTheDocument();
    expect(row.getByTitle("Run order"), "the rank and the marks share the name's line").toHaveTextContent("#1");
    expect(screen.queryByRole("note")).not.toBeInTheDocument();

    expect(within(yourComputers()).getByRole("link", { name: /^laptop/ })).toHaveTextContent("Online · 1 of 2 running");
    expect(within(yourComputers()).getByRole("link", { name: /^build-box/ })).toHaveTextContent(/seen \d+d ago/);
    await user.click(within(yourComputers()).getByRole("link", { name: /^build-box/ }));
    expect(await screen.findByRole("note")).toHaveTextContent(/Not seen for \d+ days/);
    expect(within(screen.getByRole("region", { name: "Details" })).getByText(/seen \d+d ago/), "the pane agrees with the row").toHaveClass("text-warning");
    expect(screen.getByRole("button", { name: "Remove computer" })).toBeInTheDocument();
  });

  it("opens on the first computer for an empty param, and closing takes the param away", async () => {
    const user = userEvent.setup();
    const { router } = renderComputers("/acme/agent?computers=&keep=1");

    expect(await screen.findByRole("heading", { name: "laptop", level: 3 })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(router.state.location.search).toBe("?keep=1");
  });

  it("keeps setups that never finished out of the order and removes them together after the confirm", async () => {
    const user = userEvent.setup();
    const never = { ...computer("u1", "", 0), unfinished: true, token_expires_at: "0001-01-01T00:00:00Z", server_url: "" };
    computers = [...computers, never, { ...never, id: "u2" }];
    mocks.del.mockResolvedValue({ data: undefined });
    renderComputers("/acme/agent?computers=c-laptop");

    expect(await screen.findByText(/2 setups never finished/)).toBeInTheDocument();
    expect(order()).toEqual(["laptop", "build-box"]);
    expect(screen.queryByText("#3")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Remove" }));
    await user.click(await within(await screen.findByRole("dialog", { name: /Remove these 2 setups/ })).findByRole("button", { name: "Remove" }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledTimes(2));
    expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/u1");
    expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/u2");
  });

  it("opens with focus in the list, never on Add a computer, and on the record when a deep link names one", async () => {
    renderComputers("/acme/agent?computers=c-box");
    await screen.findByRole("link", { name: /^build-box/ });
    const list = screen.getByRole("navigation", { name: "Computers and run settings" });
    expect(list.contains(document.activeElement), "not Add a computer").toBe(true);
    expect(list, "a deep link opens on its record, so the list waits behind it below 920px").toHaveClass("max-[919px]:hidden");
  });

  it("shows the list first when the dialog opens on no record, and the record once a row is picked", async () => {
    const user = userEvent.setup();
    renderComputers("/acme/agent?computers=");
    const list = await screen.findByRole("navigation", { name: "Computers and run settings" });
    expect(list).not.toHaveClass("max-[919px]:hidden");
    await screen.findByRole("region", { name: "Your computers" });
    await user.click(within(yourComputers()).getByRole("link", { name: /^build-box/ }));
    expect(list).toHaveClass("max-[919px]:hidden");
    await user.click(screen.getByRole("button", { name: "Computers" }));
    expect(list).not.toHaveClass("max-[919px]:hidden");
  });

  it("lists your order in the Defaults pane as the ranked list a project's own order is, grips included", async () => {
    mocks.put.mockResolvedValue({ data: {} });
    renderComputers("/acme/agent?computers=defaults");

    const order = await screen.findByRole("list", { name: "Your order" });
    expect(within(order).getAllByRole("listitem")).toHaveLength(2);
    expect(within(order).getByRole("button", { name: "Reorder laptop" }), "each row has its grip").toBeInTheDocument();
    expect(screen.queryByText("Drag in the list to change")).not.toBeInTheDocument();
  });

  it("words a stale computer's banner and status as one warning, and Remove asks first", async () => {
    const user = userEvent.setup();
    computers = [computers[0]!, { ...computers[1]!, runner: { connected: false, last_seen: "2026-01-01T00:00:00Z", reconnecting: false } }];
    renderComputers("/acme/agent?computers=c-box");

    await screen.findByRole("note");
    expect(within(screen.getByRole("region", { name: "Details" })).getByText(/seen \d+d ago/)).toHaveClass("text-warning");
        await user.click(screen.getByRole("button", { name: "Remove computer" }));
    expect(await screen.findByRole("dialog", { name: "Remove build-box?" })).toBeInTheDocument();
    expect(mocks.del).not.toHaveBeenCalled();
  });

  it("marks an offline computer in the Defaults list with a muted word after its name, and how long once stale", async () => {
    computers = [computers[0]!, { ...computers[1]!, runner: { connected: false, last_seen: "2026-01-01T00:00:00Z", reconnecting: false } }];
    renderComputers("/acme/agent?computers=defaults");

    const order = await screen.findByRole("list", { name: "Your order" });
    expect(within(order).getByText(/^offline \d+d$/)).toBeInTheDocument();
    expect(within(order).getAllByText(/offline/)).toHaveLength(1);
  });
});
