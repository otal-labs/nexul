import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ConversationList } from "@/components/chat/ConversationList";
import type { Conversation } from "@/models/Chat";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (err: unknown) => (err instanceof Error ? err.message : String(err)),
}));

const meResponse = {
  user: { id: "u1", provider: "github", provider_user_id: "1", login: "onik97", name: "Onik", avatar_url: "", can_create_workspace: true, first_login_done: true, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: false,
};

const conversations: Conversation[] = [
  { id: "c1", workspace_id: "ws-1", kind: "channel", name: "general", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c2", workspace_id: "ws-1", kind: "voice_channel", name: "huddle", created_by: "u1", created_at: "", updated_at: "" },
  { id: "c3", workspace_id: "ws-1", kind: "voice_channel", name: "empty-room", created_by: "u1", created_at: "", updated_at: "" },
];

const conversationsWithDocThread: Conversation[] = [
  ...conversations,
  { id: "c4", workspace_id: "ws-1", kind: "doc_thread", doc_id: "doc-1", created_by: "u1", created_at: "", updated_at: "" },
];

const docResponse = { id: "doc-1", project_id: "proj-1", title: "Runbook", body: "", version: 1, archived: false, created_at: "", updated_at: "" };

const occupancy = { c2: [{ identity: "u2", name: "Dana" }] };
const membersResponse = { members: [{ user_id: "u1", login: "onik97", role_id: "r1" }], invites: [] };

const renderList = (onSelect = vi.fn(), list: Conversation[] = conversations) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const result = render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ConversationList
        workspaceId="ws-1"
        conversations={list}
        unread={{}}
        selectedConversationId={undefined}
        onSelect={onSelect}
      />
    </QueryClientProvider>,
  );
  return { ...result, onSelect };
};

beforeEach(() => {
  useVoiceCallStore.setState({ activeConversationId: null });
  useVoiceOccupancyStore.setState({ occupancy: {} });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/voice/occupancy") return { data: occupancy };
    if (url === "/api/auth/me") return { data: meResponse };
    if (url.startsWith("/api/workspaces/")) return { data: membersResponse };
    if (url === "/api/docs/doc-1") return { data: docResponse };
    return { data: {} };
  });
});

describe("ConversationList voice channels", () => {
  it("lists voice channels in the chats group, alongside text channels", async () => {
    renderList();
    expect(await screen.findByText("general")).toBeInTheDocument();
    expect(screen.getByText("Chats")).toBeInTheDocument();
    expect(screen.getByText("huddle")).toBeInTheDocument();
    expect(screen.getByText("empty-room")).toBeInTheDocument();
  });

  it("shows occupants for an occupied voice channel, and none for an empty one", async () => {
    const { container } = renderList();
    expect(await screen.findByTitle("Dana")).toBeInTheDocument();
    // Only one occupancy chip group in the whole list — empty-room renders no avatars at all.
    expect(container.querySelectorAll('[aria-label^="In call"]')).toHaveLength(1);
  });

  it("clicking the row joins the call without opening the text view", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderList();
    await user.click(await screen.findByText("huddle"));

    expect(onSelect).not.toHaveBeenCalled();
    expect(useVoiceCallStore.getState().activeConversationId).toBe("c2");
  });

  it("clicking the text-chat affordance opens the conversation without joining the call", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderList();
    await user.click(await screen.findByLabelText("Open huddle text chat"));

    expect(onSelect).toHaveBeenCalledWith("c2");
    expect(useVoiceCallStore.getState().activeConversationId).toBeNull();
  });

  it("offers a voice channel from the new-conversation menu, distinct from the text-channel one", async () => {
    const user = userEvent.setup();
    renderList();
    await screen.findByText("Chats");

    await user.click(screen.getByRole("button", { name: "New conversation" }));
    await user.click(await screen.findByRole("menuitem", { name: "New voice channel" }));

    expect(await screen.findByLabelText("Voice channel name")).toBeInTheDocument();
    expect(screen.queryByLabelText("Channel name")).not.toBeInTheDocument();
  });
});

describe("ConversationList doc threads", () => {
  it("shows an empty threads group when the caller has none", async () => {
    renderList();
    await screen.findByText("general");
    expect(screen.getByText("Threads")).toBeInTheDocument();
    expect(screen.getByText("No threads yet")).toBeInTheDocument();
  });

  it("lists a doc thread under the threads group, titled with the doc", async () => {
    renderList(vi.fn(), conversationsWithDocThread);
    expect(await screen.findByText("Threads")).toBeInTheDocument();
    expect(await screen.findByText("Runbook")).toBeInTheDocument();
  });

  it("opens the doc thread's conversation on click", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderList(vi.fn(), conversationsWithDocThread);
    await user.click(await screen.findByText("Runbook"));
    expect(onSelect).toHaveBeenCalledWith("c4");
  });
});
