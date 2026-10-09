import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { AxiosRequestConfig } from "axios";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import type { Botwebhook } from "@/models/Botwebhook";
import type { Conversation } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: (error: unknown) => (error as Error).message,
}));
vi.mock("sonner", () => ({ toast: mocks.toast }));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "alice", name: "Alice", avatar_url: "", first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const people = [
  { user_id: "u1", login: "alice", display_name: "Alice", avatar_url: "" },
  { user_id: "u2", login: "bob", display_name: "Bob", avatar_url: "" },
];

const conversation = (overrides: Partial<Conversation>): Conversation => ({
  id: "c5",
  workspace_id: "ws-1",
  kind: "channel",
  name: "eng",
  created_by: "u1",
  created_at: "",
  updated_at: "",
  ...overrides,
});

const bot = (id: string, name: string, overrides: Partial<Botwebhook> = {}): Botwebhook => ({
  id,
  conversation_id: "c5",
  name,
  created_by: "u2",
  created_at: new Date(Date.now() - 3 * 86_400_000).toISOString(),
  updated_at: "",
  post_count: 0,
  url: `https://nexul.example.com/api/botwebhooks/${id}/token-${id}`,
  ...overrides,
});

let conversations: Conversation[] = [];
let permissions: string[] = [];
let live: Botwebhook[] = [];
let deleted: Botwebhook[] = [];

// A conversation's bots as the gateway keeps them, so a mutation's refetch reads its own result.
const serveBots = () => {
  vi.mocked(api.post).mockImplementation(async (_url: string, body: unknown) => {
    const created = bot(`b${live.length + deleted.length + 1}`, (body as { name: string }).name);
    live = [...live, created];
    return { data: created };
  });
  vi.mocked(api.patch).mockImplementation(async (url: string, body: unknown) => {
    const id = url.split("/").pop();
    const change = body as { regenerate?: boolean; deleted?: boolean };
    const restored = deleted.find((b) => b.id === id);
    if (change.deleted === false && restored) {
      deleted = deleted.filter((b) => b !== restored);
      live = [...live, bot(restored.id, restored.name, { url: `${restored.url}-new` })];
    }
    if (change.regenerate) live = live.map((b) => (b.id === id ? { ...b, url: `${b.url}-new` } : b));
    return { data: live.find((b) => b.id === id) };
  });
  vi.mocked(api.delete).mockImplementation(async (url: string) => {
    const id = url.split("/").pop();
    live = live.filter((b) => b.id !== id);
    return { data: undefined };
  });
};

const renderSidebar = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/acme/chat/c5"]}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ChatSidebarSection collapsed={false} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const openMenuItem = async (u: ReturnType<typeof userEvent.setup>, label: string, item: string) => {
  await u.click(await screen.findByRole("button", { name: `More actions for ${label}` }));
  await u.click(await screen.findByRole("menuitem", { name: item }));
  return screen.findByRole("dialog", { name: label });
};

const openSettings = (u: ReturnType<typeof userEvent.setup>) => openMenuItem(u, "#eng", "Settings");

beforeEach(() => {
  conversations = [conversation({})];
  permissions = ["chat:write", "botwebhook:read", "botwebhook:write", "botwebhook:delete"];
  live = [bot("b1", "CI", { last_post_at: new Date(Date.now() - 24 * 60_000).toISOString() }), bot("b2", "Alerts")];
  deleted = [];
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  for (const fn of Object.values(api)) vi.mocked(fn).mockReset();
  for (const fn of Object.values(mocks.toast)) fn.mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string, config?: AxiosRequestConfig) => {
    if (url === "/api/chat/conversations") return { data: conversations };
    if (url === "/api/chat/unread") return { data: {} };
    if (url === "/api/auth/me") return { data: meResponse };
    if (url === "/api/voice/occupancy") return { data: {} };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/workspaces/ws-1/people") return { data: { people } };
    if (url.endsWith("/botwebhooks")) return { data: (config?.params as { deleted?: boolean } | undefined)?.deleted ? deleted : live };
    return { data: {} };
  });
  serveBots();
});

describe("the Bots section", () => {
  it("opens settings on a public channel for someone who only reads bots, and shows them rows without URLs or editing", async () => {
    permissions = ["chat:write", "botwebhook:read"];
    live = live.map((b) => ({ ...b, url: "" }));
    deleted = [bot("b9", "Old hook", { deleted_at: new Date().toISOString(), deleted_by: "u2" })];
    renderSidebar();
    const dialog = await openSettings(userEvent.setup());

    const rows = await within(dialog).findByRole("list", { name: "Bots" });
    expect(within(rows).getByText("CI")).toBeInTheDocument();
    expect(within(rows).getByText("Last post 24m ago · made by Bob")).toBeInTheDocument();
    expect(within(rows).getByText("No posts yet · made by Bob")).toBeInTheDocument();
    expect(within(dialog).getByText("2 of 10")).toBeInTheDocument();
    expect(within(rows).queryByRole("button")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "New bot" })).not.toBeInTheDocument();
    expect(within(dialog).queryByText(/Deleted \(/)).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/conversations/c5/botwebhooks", { params: { deleted: true } });
  });

  it("creates a bot and lands on it with its URL", async () => {
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: "New bot" }));
    await u.type(within(dialog).getByRole("textbox", { name: "Name" }), "Uptime");
    await u.click(within(dialog).getByRole("button", { name: "Create bot" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/conversations/c5/botwebhooks", { name: "Uptime", avatar: "" }));
    expect(await within(dialog).findByRole("textbox", { name: "Uptime's webhook URL" })).toHaveValue(
      "https://nexul.example.com/api/botwebhooks/b3/token-b3",
    );
  });

  it("refuses a name another bot here has as it is typed, and shows the server's refusal inline", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("this conversation already has a bot named Deploys"));
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: "New bot" }));
    const name = within(dialog).getByRole("textbox", { name: "Name" });
    await u.type(name, "ci");
    expect(await within(dialog).findByText("Another bot here is called ci.")).toBeInTheDocument();

    await u.clear(name);
    await u.type(name, "Deploys");
    await u.click(within(dialog).getByRole("button", { name: "Create bot" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("this conversation already has a bot named Deploys");
    expect(within(dialog).getByRole("textbox", { name: "Name" })).toHaveValue("Deploys");
  });

  it("holds ten bots: New bot and Restore are off and say why", async () => {
    live = Array.from({ length: 10 }, (_, i) => bot(`b${i + 1}`, `Bot ${i + 1}`));
    deleted = [bot("b11", "Old hook", { deleted_at: new Date().toISOString(), deleted_by: "u2" })];
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);

    expect(await within(dialog).findByRole("button", { name: "New bot" })).toBeDisabled();
    expect(within(dialog).getByText("A channel holds ten bots. Delete one to add another.")).toBeInTheDocument();
    await u.click(await within(dialog).findByRole("button", { name: "Deleted (1)" }));
    expect(within(dialog).getByRole("button", { name: "Restore Old hook" })).toBeDisabled();
  });

  it("regenerates only after the confirm that names the 404, then says the old URL stopped working", async () => {
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: /^CI/ }));
    await u.click(within(dialog).getByRole("button", { name: "Regenerate URL" }));

    const confirm = await screen.findByRole("dialog", { name: "Regenerate CI's URL?" });
    expect(confirm).toHaveTextContent("Anything still posting to the old URL gets a 404 until it uses the new one.");
    await u.click(within(confirm).getByRole("button", { name: "Regenerate URL" }));

    await waitFor(() => expect(api.patch).toHaveBeenCalledWith("/api/botwebhooks/b1", { regenerate: true }));
    expect(await within(dialog).findByText("New URL. The old one no longer works.")).toBeInTheDocument();
    expect(within(dialog).getByRole("textbox", { name: "CI's webhook URL" })).toHaveValue(
      "https://nexul.example.com/api/botwebhooks/b1/token-b1-new",
    );
  });

  it("deletes only after the confirm and goes back to the list", async () => {
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: /^CI/ }));
    await u.click(within(dialog).getByRole("button", { name: "Delete bot" }));

    const confirm = await screen.findByRole("dialog", { name: "Delete CI?" });
    expect(confirm).toHaveTextContent("you can restore it later with a new URL.");
    await u.click(within(confirm).getByRole("button", { name: "Delete bot" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/botwebhooks/b1"));
    expect(await within(dialog).findByText("1 of 10")).toBeInTheDocument();
  });

  it("hides Delete bot from someone who may edit bots but not delete them", async () => {
    permissions = ["chat:write", "botwebhook:read", "botwebhook:write"];
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: /^CI/ }));

    expect(within(dialog).getByRole("button", { name: "Regenerate URL" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Delete bot" })).not.toBeInTheDocument();
  });

  it("restores a deleted bot from the fold and lands on it with its new URL", async () => {
    deleted = [bot("b9", "Old hook", { deleted_at: new Date().toISOString(), deleted_by: "u2" })];
    const u = userEvent.setup();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(await within(dialog).findByRole("button", { name: "Deleted (1)" }));
    expect(within(dialog).getByText("Deleted just now by Bob")).toBeInTheDocument();
    await u.click(within(dialog).getByRole("button", { name: "Restore Old hook" }));

    await waitFor(() => expect(api.patch).toHaveBeenCalledWith("/api/botwebhooks/b9", { deleted: false }));
    expect(await within(dialog).findByText("Restored with a new URL.")).toBeInTheDocument();
  });

  it("opens a DM's bots from its menu", async () => {
    conversations = [{ id: "dm1", workspace_id: "ws-1", kind: "dm", participant_ids: ["u1", "u2"], created_by: "u1", created_at: "", updated_at: "" }];
    renderSidebar();
    const dialog = await openMenuItem(userEvent.setup(), "Bob", "Bots");

    expect(await within(dialog).findByRole("list", { name: "Bots" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/conversations/dm1/botwebhooks", { params: {} });
  });
});
