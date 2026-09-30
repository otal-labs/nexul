import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (error: unknown) => String(error),
}));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversations = [
  { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c2", workspace_id: "ws-1", kind: "dm", created_by: "u1", created_at: "", updated_at: "", participant_ids: ["u1", "u2"] },
  { id: "c3", workspace_id: "ws-1", kind: "voice_channel", name: "huddle", created_by: "u1", created_at: "", updated_at: "" },
];

const occupancy = { c3: [{ identity: "u2", name: "Dana" }] };

const peopleResponse = {
  people: [
    { user_id: "u1", login: "onik97", display_name: "", avatar_url: "" },
    { user_id: "u2", login: "olive", display_name: "", avatar_url: "" },
  ],
};

const renderSection = (collapsed = false, path = "/") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ChatSidebarSection collapsed={collapsed} />
        <Routes>
          <Route path="/" element={null} />
          <Route path="/acme/chat/:conversationId" element={<div>chat-page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useVoiceCallStore.setState({ activeConversationId: null });
  useVoiceOccupancyStore.setState({ occupancy: {} });
  localStorage.clear();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/chat/conversations") return { data: conversations };
    if (url === "/api/chat/unread") return { data: { c1: 7 } };
    if (url === "/api/auth/me") return { data: meResponse };
    if (url === "/api/voice/occupancy") return { data: occupancy };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["chat:write"] } };
    if (url.startsWith("/api/workspaces/")) return { data: peopleResponse };
    return { data: {} };
  });
});

describe("ChatSidebarSection", () => {
  it("renders nothing while collapsed", () => {
    const { container } = renderSection(true);
    expect(container).toBeEmptyDOMElement();
  });

  it("lists channels and DMs under their own headings, with unread badges", async () => {
    renderSection();
    expect(await screen.findByText("general")).toBeInTheDocument();
    expect(screen.getByText("Channels")).toBeInTheDocument();
    expect(screen.getByText("Direct messages")).toBeInTheDocument();
    expect(screen.getByText("olive")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
  });

  it("with no conversations it still offers to create each kind", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") return { data: [] };
      if (url === "/api/auth/me") return { data: meResponse };
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["chat:write"] } };
      return { data: {} };
    });
    renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/chat/conversations", expect.anything()));
    expect(await screen.findByRole("button", { name: "New channel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New voice channel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "New direct message" })).toBeInTheDocument();
  });

  it("the voice channels + creates one without leaving the page", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "c9", workspace_id: "ws-1", kind: "voice_channel", name: "standup", created_by: "u1", created_at: "", updated_at: "" },
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "New voice channel" }));
    await user.type(await screen.findByLabelText("Voice channel name"), "standup");
    await user.click(screen.getByRole("button", { name: "Create voice channel" }));

    await waitFor(() => expect(api.post).toHaveBeenCalled());
    expect(vi.mocked(api.post).mock.calls[0]?.[1]).toEqual({ workspace_id: "ws-1", name: "standup" });
    expect(screen.queryByText("chat-page")).not.toBeInTheDocument();
  });

  it("the channels + opens the new channel", async () => {
    vi.mocked(api.post).mockResolvedValue({
      data: { id: "c8", workspace_id: "ws-1", kind: "channel", name: "incidents", created_by: "u1", created_at: "", updated_at: "" },
    });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "New channel" }));
    await user.type(await screen.findByLabelText("Channel name"), "incidents");
    await user.click(screen.getByRole("button", { name: "Create channel" }));

    expect(await screen.findByText("chat-page")).toBeInTheDocument();
    expect(api.post).toHaveBeenCalledWith("/api/chat/channels", { workspace_id: "ws-1", name: "incidents" });
  });

  it("clicking a conversation navigates to its chat page", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByText("general"));
    expect(await screen.findByText("chat-page")).toBeInTheDocument();
  });

  it("lists voice channels under their own heading, with occupants visible without joining", async () => {
    renderSection();
    expect(await screen.findByText("huddle")).toBeInTheDocument();
    expect(screen.getByText("Voice channels")).toBeInTheDocument();
    expect(await screen.findByTitle("Dana")).toBeInTheDocument();
  });

  it("clicking a voice channel joins the call and opens its text chat", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByText("huddle"));

    expect(await screen.findByText("chat-page")).toBeInTheDocument();
    expect(useVoiceCallStore.getState().activeConversationId).toBe("c3");
  });

  it("lists doc threads under Threads, titled with their doc", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") {
        return { data: [{ id: "c4", workspace_id: "ws-1", kind: "doc_thread", doc_id: "doc-1", created_by: "u1", created_at: "", updated_at: "" }] };
      }
      if (url === "/api/docs/doc-1") return { data: { id: "doc-1", title: "Runbook" } };
      if (url === "/api/auth/me") return { data: meResponse };
      return { data: {} };
    });
    renderSection();
    expect(await screen.findByText("Runbook")).toBeInTheDocument();
    expect(screen.getByText("Threads")).toBeInTheDocument();
  });

  it("marks the open conversation's row as the current page", async () => {
    renderSection(false, "/acme/chat/c1");
    expect(await screen.findByRole("link", { name: /general/ })).toHaveAttribute("aria-current", "page");
  });
});
