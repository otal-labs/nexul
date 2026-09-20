import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ChatPage } from "@/pages/ChatPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
}));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", can_create_workspace: true, first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversations = [
  { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c2", workspace_id: "ws-1", kind: "doc_thread", doc_id: "doc-1", created_by: "u1", created_at: "", updated_at: "" },
];

const docResponse = { id: "doc-1", project_id: "proj-1", title: "Runbook", body: "", version: 1, archived: false, created_at: "", updated_at: "" };

const renderPage = (path = "/chat") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/chat" element={<ChatPage />} />
          <Route path="/chat/:conversationId" element={<ChatPage />} />
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
    if (url === "/api/docs/doc-1") return { data: docResponse };
    if (url.startsWith("/api/chat/conversations/")) return { data: [] };
    return { data: { members: [], invites: [] } };
  });
});

describe("ChatPage", () => {
  it("groups the list into chats and threads and prompts to pick one before a thread is open", async () => {
    renderPage();
    expect(await screen.findByText("general")).toBeInTheDocument();
    expect(screen.getByText("Chats")).toBeInTheDocument();
    expect(screen.getByText("Threads")).toBeInTheDocument();
    expect(await screen.findByText("Runbook")).toBeInTheDocument();
    expect(screen.getByText("Select a conversation")).toBeInTheDocument();
  });

  it("opens the conversation named in the URL with its composer", async () => {
    renderPage("/chat/c1");
    expect(await screen.findByLabelText("Message")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "general" })).toHaveAttribute("aria-current", "true");
  });

  it("navigates to the picked conversation and back to the list", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByText("general"));
    await screen.findByLabelText("Message");
    await user.click(screen.getByRole("button", { name: "Back to conversations" }));
    expect(await screen.findByText("Select a conversation")).toBeInTheDocument();
  });
});
