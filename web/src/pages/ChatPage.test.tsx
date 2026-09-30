import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatPage } from "@/pages/ChatPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversations = [
  { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c2", workspace_id: "ws-1", kind: "doc_thread", doc_id: "doc-1", created_by: "u1", created_at: "", updated_at: "" },
];

const renderPage = (path = "/acme/chat") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/acme/chat" element={<ChatPage />} />
          <Route path="/acme/chat/:conversationId" element={<ChatPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/chat/conversations") return { data: conversations };
    if (url === "/api/chat/unread") return { data: {} };
    if (url === "/api/auth/me") return { data: meResponse };
    if (url.startsWith("/api/chat/conversations/")) return { data: [] };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [] } };
    return { data: {} };
  });
});

describe("ChatPage", () => {
  it("a bare /chat opens the first channel", async () => {
    renderPage();
    expect(await screen.findByLabelText("Message")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Message general…")).toBeInTheDocument();
  });

  it("opens the conversation named in the URL with its composer", async () => {
    renderPage("/acme/chat/c1");
    expect(await screen.findByPlaceholderText("Message general…")).toBeInTheDocument();
  });

  it("says so when the URL names a conversation that is not in the workspace", async () => {
    renderPage("/acme/chat/gone");
    expect(await screen.findByText("Conversation not found")).toBeInTheDocument();
  });

  it("a workspace with no conversations shows the empty state instead of redirecting", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") return { data: [] };
      return { data: {} };
    });
    renderPage();
    expect(await screen.findByText("No conversations yet")).toBeInTheDocument();
  });
});

// The reported bug: a member without members:write saw the Owner's raw user id as the DM's name and the message author.
describe("ChatPage for a member without members:write", () => {
  const dm = { id: "c3", workspace_id: "ws-1", kind: "dm", participant_ids: ["u1", "u-lewis"], created_by: "u-lewis", created_at: "", updated_at: "" };
  const sup = { id: "m1", conversation_id: "c3", author_id: "u-lewis", author_kind: "user", body: "Sup man", mentions: null, created_at: "2026-09-29T00:00:00Z", updated_at: "2026-09-29T00:00:00Z" };

  const stagePeople = (displayName: string) => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") return { data: [dm] };
      if (url === "/api/chat/unread") return { data: {} };
      if (url === "/api/auth/me") return { data: meResponse };
      if (url.startsWith("/api/chat/conversations/c3/messages")) return { data: [sup] };
      if (url === "/api/workspaces/ws-1/members") throw Object.assign(new Error("forbidden"), { response: { status: 403 } });
      if (url === "/api/workspaces/ws-1/people") {
        return {
          data: {
            people: [
              { user_id: "u1", login: "onik97", display_name: "Onik", avatar_url: "" },
              { user_id: "u-lewis", login: "LewisWelch94", display_name: displayName, avatar_url: "" },
            ],
          },
        };
      }
      return { data: {} };
    });
  };

  it("names the DM and its author by their display name", async () => {
    stagePeople("Lewis");
    renderPage("/acme/chat/c3");
    expect(await screen.findByText("Sup man")).toBeInTheDocument();
    // The thread header and the message author.
    expect(await screen.findAllByText("Lewis")).toHaveLength(2);
    expect(screen.queryByText("u-lewis")).not.toBeInTheDocument();
  });

  it("falls back to the login when they have no display name", async () => {
    stagePeople("");
    renderPage("/acme/chat/c3");
    expect(await screen.findByText("Sup man")).toBeInTheDocument();
    // The thread header and the message author.
    expect(await screen.findAllByText("LewisWelch94")).toHaveLength(2);
    expect(screen.queryByText("u-lewis")).not.toBeInTheDocument();
  });
});
