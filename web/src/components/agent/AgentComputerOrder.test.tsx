import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Computer } from "@/models/Pairing";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { renderAgent } from "@/test/agentComputers";
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
  if (url === "/api/pairing/presence") return { computers: {} };
  if (url === "/api/runners/enrollments") return [];
  if (url === "/api/pairing/defaults") return {};
  if (url === "/api/pairing/projects") return { links: [] };
  if (url === "/api/projects") return [];
  if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme", slug: "acme" }];
  if (url.endsWith("/setup")) return { computer_id: "", confirmed_at: "2026-10-01T00:00:00Z", providers: [], turns: [], skipped_providers: [], models: {}, model_options: {}, folder: "" };
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
    renderAgent("/acme/agent/computers/c-laptop");

    expect(await screen.findByText("#1")).toBeInTheDocument();
    expect(screen.getByText("#2")).toBeInTheDocument();
    expect(order()).toEqual(["laptop", "build-box"]);
    expect(await screen.findByText("1 of 2 play runs")).toBeInTheDocument();

    await user.click(within(yourComputers()).getByRole("link", { name: /build-box$/ }));
    expect(await screen.findByText("0 of 2 play runs")).toBeInTheDocument();
  });

  it("moves a computer down from its menu, at once, and sends the new order", async () => {
    const user = userEvent.setup();
    mocks.put.mockImplementation(async () => {
      computers = [{ ...computers[1]!, position: 0 }, { ...computers[0]!, position: 1 }];
      return { data: {} };
    });
    renderAgent("/acme/agent/computers/c-laptop");

    await user.click(await screen.findByRole("button", { name: "Actions for laptop" }));
    await user.click(await screen.findByRole("button", { name: "Move down" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/order", { computer_ids: ["c-box", "c-laptop"] });
    await waitFor(() => expect(order()).toEqual(["build-box", "laptop"]));
  });

  it("puts the order back when the server refuses it", async () => {
    const user = userEvent.setup();
    mocks.put.mockRejectedValue(new Error("the order must name each of your computers once"));
    renderAgent("/acme/agent/computers/c-box");

    await user.click(await screen.findByRole("button", { name: "Actions for build-box" }));
    await user.click(await screen.findByRole("button", { name: "Move up" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalled());
    await waitFor(() => expect(order()).toEqual(["laptop", "build-box"]));
  });

  it("sets how many runs a computer takes at once from its details", async () => {
    const user = userEvent.setup();
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderAgent("/acme/agent/computers/c-laptop");

    await user.click(within(await screen.findByRole("radiogroup", { name: "Runs at once on laptop" })).getByRole("radio", { name: "4" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { runs_at_once: 4 });
    expect(await screen.findByText("1 of 4 play runs")).toBeInTheDocument();
  });

  it("switches a computer off for plays from its details, saying what still works there", async () => {
    const user = userEvent.setup();
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderAgent("/acme/agent/computers/c-laptop");

    const takes = await screen.findByRole("switch", { name: "Plays, auto plays and @Agent run here" });
    expect(takes).toBeChecked();
    await user.click(takes);

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { plays_off: true });
    expect(await screen.findByText("Commands, logs and setup still work.")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Plays, auto plays and @Agent run here" })).not.toBeChecked();
  });
});
