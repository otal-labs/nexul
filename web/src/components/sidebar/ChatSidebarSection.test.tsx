import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", can_create_workspace: true, first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversations = [
  { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c2", workspace_id: "ws-1", kind: "dm", created_by: "u1", created_at: "", updated_at: "", participant_ids: ["u1", "u2"] },
  { id: "c3", workspace_id: "ws-1", kind: "voice_channel", name: "huddle", created_by: "u1", created_at: "", updated_at: "" },
];

const occupancy = { c3: [{ identity: "u2", name: "Dana" }] };

const membersResponse = {
  members: [
    { user_id: "u1", login: "onik97", role_id: "r1" },
    { user_id: "u2", login: "olive", role_id: "r1" },
  ],
  invites: [],
};

const renderSection = (collapsed = false) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<ChatSidebarSection collapsed={collapsed} />} />
          <Route path="/chat/:conversationId" element={<div>chat-page</div>} />
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
    if (url.startsWith("/api/workspaces/")) return { data: membersResponse };
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

  it("renders nothing when there are no conversations", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") return { data: [] };
      if (url === "/api/auth/me") return { data: meResponse };
      return { data: {} };
    });
    const { container } = renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/chat/conversations", expect.anything()));
    expect(container).toBeEmptyDOMElement();
  });

  it("clicking a conversation navigates to its chat page", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByText("general"));
    expect(await screen.findByText("chat-page")).toBeInTheDocument();
  });

  it("lists voice channels under their own heading, with occupants visible without joining", async () => {
    renderSection();
    expect(await screen.findByText("Voice channels")).toBeInTheDocument();
    expect(screen.getByText("huddle")).toBeInTheDocument();
    expect(await screen.findByTitle("Dana")).toBeInTheDocument();
  });

  it("clicking a voice channel joins the call without leaving the page", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByText("huddle"));

    expect(screen.queryByText("chat-page")).not.toBeInTheDocument();
    expect(useVoiceCallStore.getState().activeConversationId).toBe("c3");
  });
});
