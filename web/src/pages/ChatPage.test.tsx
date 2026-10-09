import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { upsertCachedMessage } from "@/hooks/ChatHooks";
import type { Message } from "@/models/Chat";
import { ChatPage } from "@/pages/ChatPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

// Every message row asks for the delete confirmation hook once per render, so counting its calls counts row renders.
const rows = vi.hoisted(() => ({ renders: 0 }));
vi.mock("@/hooks/useConfirmationDialog", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/useConfirmationDialog")>("@/hooks/useConfirmationDialog");
  return {
    ...actual,
    useConfirmationDialog: () => {
      rows.renders += 1;
      return actual.useConfirmationDialog();
    },
  };
});

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
  const view = render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/acme/chat" element={<ChatPage />} />
          <Route path="/acme/chat/:conversationId" element={<ChatPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, client };
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

describe("ChatPage live messages", () => {
  const history: Message[] = Array.from({ length: 6 }, (_, i) => ({
    id: `m${i}`,
    conversation_id: "c1",
    author_id: i % 2 === 0 ? "u1" : "u2",
    author_kind: "user",
    body: `Message ${i}`,
    mentions: null,
    created_at: `2026-09-29T00:0${i}:00Z`,
    updated_at: `2026-09-29T00:0${i}:00Z`,
  }));
  // A bot is no one the people directory knows, so its author is looked up as an unknown person on every render.
  history.push({ ...history[0]!, id: "m-bot", author_id: "bot-1", author_kind: "bot", author_name: "Deploys", body: "Deployed api 1.4.0" });

  it("renders only the arriving message's row when a message lands in an open conversation", async () => {
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/chat/conversations") return { data: conversations };
      if (url === "/api/chat/unread") return { data: {} };
      if (url === "/api/auth/me") return { data: meResponse };
      if (url.startsWith("/api/chat/conversations/c1/messages")) return { data: history };
      if (url === "/api/workspaces/ws-1/people") {
        return { data: { people: [{ user_id: "u1", login: "onik97", display_name: "Onik", avatar_url: "" }, { user_id: "u2", login: "lena", display_name: "Lena", avatar_url: "" }] } };
      }
      return { data: {} };
    });
    vi.mocked(api.post).mockResolvedValue({ data: {} });
    const { client } = renderPage("/acme/chat/c1");
    expect(await screen.findByText("Message 5")).toBeInTheDocument();
    expect(await screen.findAllByText("Lena")).not.toHaveLength(0);

    rows.renders = 0;
    await act(async () => {
      upsertCachedMessage(client, { ...history[1]!, id: "m6", body: "Message 6", created_at: "2026-09-29T00:06:00Z", updated_at: "2026-09-29T00:06:00Z" });
    });

    expect(await screen.findByText("Message 6")).toBeInTheDocument();
    expect(rows.renders).toBe(1);
  });
});
