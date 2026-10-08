import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import type { Conversation } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: (error: unknown) => (error as Error).message,
}));
vi.mock("sonner", () => ({ toast: mocks.toast }));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik", name: "Onik", avatar_url: "", first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const people = [
  { user_id: "u1", login: "onik", display_name: "Onik", avatar_url: "" },
  { user_id: "u2", login: "bob", display_name: "Bob", avatar_url: "" },
  { user_id: "u3", login: "sam", display_name: "Sam", avatar_url: "" },
];

const channel = (overrides: Partial<Conversation>): Conversation => ({
  id: "c5",
  workspace_id: "ws-1",
  kind: "channel",
  name: "eng",
  created_by: "u1",
  created_at: "",
  updated_at: "",
  ...overrides,
});

let conversations: Conversation[] = [];
let permissions: string[] = [];
let restricted = false;

const renderSidebar = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/acme/chat/c5"]}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ChatSidebarSection collapsed={false} />
        <Routes>
          <Route path="/acme/chat" element={<div>chat-home</div>} />
          <Route path="/acme/chat/dm-bob" element={<div>dm-with-bob</div>} />
          <Route path="/acme/chat/:conversationId" element={<div>chat-page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const user = () => userEvent.setup();

const openSettings = async (u: ReturnType<typeof userEvent.setup>, label = "#eng") => {
  await u.click(await screen.findByRole("button", { name: `More actions for ${label}` }));
  await u.click(await screen.findByRole("menuitem", { name: "Settings" }));
  return screen.findByRole("dialog", { name: label });
};

beforeEach(() => {
  conversations = [channel({})];
  permissions = ["chat:write", "channels:write"];
  restricted = false;
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  for (const fn of Object.values(api)) vi.mocked(fn).mockReset();
  for (const fn of Object.values(mocks.toast)) fn.mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/chat/conversations") return { data: conversations };
    if (url === "/api/chat/unread") return { data: {} };
    if (url === "/api/auth/me") return { data: meResponse };
    if (url === "/api/voice/occupancy") return { data: {} };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions, restricted } };
    if (url === "/api/workspaces/ws-1/people") return { data: { people } };
    return { data: {} };
  });
});

describe("a channel's settings", () => {
  it("shows the server's refusal when making a channel public fails, and the channel stays private", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    vi.mocked(api.put).mockRejectedValue(new Error("a member who sees only some projects may only have private channels"));
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("switch", { name: "Private channel" }));
    await u.click(await screen.findByRole("button", { name: "Make public" }));

    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("a member who sees only some projects may only have private channels"));
    expect(within(dialog).getByRole("switch", { name: "Private channel" })).toBeChecked();
  });

  it("offers no Leave to the last member", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1"] })];
    renderSidebar();
    const dialog = await openSettings(user());
    expect(within(dialog).getByText("1 member")).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Leave channel" })).not.toBeInTheDocument();
  });

  it("lets a member without channels:write add people and leave, but neither switch nor remove", async () => {
    permissions = ["chat:write"];
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    expect(within(dialog).queryByRole("switch")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Add people" })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Leave channel" })).toBeInTheDocument();
    await u.click(within(dialog).getByRole("button", { name: "Actions for Bob" }));
    expect((await screen.findAllByRole("menuitem")).map((item) => item.textContent)).toEqual(["Send a message"]);
  });

  it("offers no Settings on a public channel to someone who cannot switch it", async () => {
    permissions = ["chat:write"];
    renderSidebar();
    expect(await screen.findByText("eng")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for #eng" })).not.toBeInTheDocument();
  });

  it("holds a Restricted member's private channel private", async () => {
    restricted = true;
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    renderSidebar();
    const dialog = await openSettings(user());
    await waitFor(() => expect(within(dialog).getByRole("switch", { name: "Private channel" })).toBeDisabled());
  });

  it("turns private keeping the switcher and whoever is checked", async () => {
    vi.mocked(api.put).mockResolvedValue({ data: channel({ private: true, participant_ids: ["u1", "u3"] }) });
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("switch", { name: "Private channel" }));

    const picker = await screen.findByRole("dialog", { name: "Who stays in #eng?" });
    expect(within(picker).getByRole("checkbox", { name: "Onik" })).toBeChecked();
    expect(within(picker).getByRole("checkbox", { name: "Onik" })).toBeDisabled();
    expect(within(picker).getByText("1 of 3 people")).toBeInTheDocument();
    await u.type(within(picker).getByRole("textbox", { name: "Search people" }), "sa");
    expect(within(picker).queryByRole("checkbox", { name: "Bob" })).not.toBeInTheDocument();
    await u.click(within(picker).getByRole("checkbox", { name: "Sam" }));
    await u.click(within(picker).getByRole("button", { name: "Make private" }));

    await waitFor(() => expect(api.put).toHaveBeenCalledWith("/api/chat/conversations/c5/private", { private: true, member_ids: ["u3"] }));
    expect(mocks.toast.success).toHaveBeenCalledWith("#eng is now private");
  });

  it("turns public only after a confirmation that names the history", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    vi.mocked(api.put).mockResolvedValue({ data: channel({ private: false }) });
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("switch", { name: "Private channel" }));
    expect(await screen.findByText("Everyone in the workspace will see it and read its whole history.")).toBeInTheDocument();
    await u.click(screen.getByRole("button", { name: "Make public" }));

    await waitFor(() => expect(api.put).toHaveBeenCalledWith("/api/chat/conversations/c5/private", { private: false, member_ids: [] }));
    expect(mocks.toast.success).toHaveBeenCalledWith("#eng is now public");
  });

  it("adds people from those not yet in it", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    vi.mocked(api.post).mockResolvedValue({ data: channel({ private: true, participant_ids: ["u1", "u2", "u3"] }) });
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("button", { name: "Add people" }));

    const form = await screen.findByRole("dialog", { name: "Add people to #eng" });
    expect(within(form).queryByRole("checkbox", { name: "Bob" })).not.toBeInTheDocument();
    await u.click(within(form).getByRole("checkbox", { name: "Sam" }));
    await u.click(within(form).getByRole("button", { name: "Add people" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/chat/conversations/c5/members", { user_ids: ["u3"] }));
    expect(mocks.toast.success).toHaveBeenCalledWith("Added 1 person to #eng");
  });

  it("removes someone from the row's menu", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    vi.mocked(api.delete).mockResolvedValue({ data: channel({ private: true, participant_ids: ["u1"] }) });
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("button", { name: "Actions for Bob" }));
    await u.click(await screen.findByRole("menuitem", { name: "Remove from channel" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/chat/conversations/c5/members/u2"));
    expect(mocks.toast.success).toHaveBeenCalledWith("Removed Bob from #eng");
  });

  it("leaves after a confirmation and sends a viewer of the channel to the chat home", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] })];
    vi.mocked(api.post).mockResolvedValue({ data: channel({ private: true, participant_ids: ["u2"] }) });
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("button", { name: "Leave channel" }));
    await u.click(await screen.findByRole("button", { name: "Leave" }));

    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/chat/conversations/c5/leave"));
    expect(await screen.findByText("chat-home")).toBeInTheDocument();
    expect(mocks.toast.success).toHaveBeenCalledWith("You left #eng");
  });

  it("sends a message through the existing DM and closes the settings", async () => {
    const dm: Conversation = { id: "dm-bob", workspace_id: "ws-1", kind: "dm", created_by: "u1", created_at: "", updated_at: "", participant_ids: ["u2", "u1"] };
    conversations = [channel({ private: true, participant_ids: ["u1", "u2"] }), dm];
    const u = user();
    renderSidebar();
    const dialog = await openSettings(u);
    await u.click(within(dialog).getByRole("button", { name: "Actions for Bob" }));
    await u.click(await screen.findByRole("menuitem", { name: "Send a message" }));

    expect(await screen.findByText("dm-with-bob")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(api.post).not.toHaveBeenCalled();
  });

  it("marks a private channel in the sidebar with a lock", async () => {
    conversations = [channel({ private: true, participant_ids: ["u1"] }), channel({ id: "c6", name: "ops" })];
    renderSidebar();
    expect(await screen.findByRole("link", { name: /eng/ })).toContainElement(screen.getByRole("img", { name: "Private" }));
    expect(screen.getAllByRole("img", { name: "Private" })).toHaveLength(1);
  });
});

describe("creating a channel", () => {
  it("creates it private with the people picked", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: channel({ id: "c9", name: "client", private: true }) });
    const u = user();
    renderSidebar();
    await u.click(await screen.findByRole("button", { name: "New channel" }));
    await u.type(await screen.findByLabelText("Channel name"), "client");
    await u.click(screen.getByRole("switch", { name: "Private channel" }));
    await u.click(await screen.findByRole("checkbox", { name: "Bob" }));
    await u.click(screen.getByRole("button", { name: "Create channel" }));

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/chat/channels", { workspace_id: "ws-1", name: "client", private: true, member_ids: ["u2"] }),
    );
    expect(mocks.toast.success).toHaveBeenCalledWith("#client created");
  });

  it("gives a Restricted member only a private channel, its switch fixed on", async () => {
    restricted = true;
    const u = user();
    renderSidebar();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await u.click(await screen.findByRole("button", { name: "New channel" }));
    const toggle = await screen.findByRole("switch", { name: "Private channel" });
    expect(toggle).toBeChecked();
    expect(toggle).toBeDisabled();
  });
});
