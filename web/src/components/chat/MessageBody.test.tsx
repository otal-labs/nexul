import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import { MessageBody } from "@/components/chat/MessageBody";
import { getChatConversationsKey } from "@/hooks/ChatHooks";
import type { Conversation } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const channel: Conversation = {
  id: "c-ops",
  workspace_id: "ws-1",
  kind: "channel",
  name: "ops",
  created_by: "u1",
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
};

const renderBody = (body: string, seed?: (client: QueryClient) => void) => {
  const client = new QueryClient();
  seed?.(client);
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/fahad/chat/c-general"]}>
        <Routes>
          <Route path="/fahad/chat/c-general" element={<MessageBody body={body} mentionHandles={["lena"]} />} />
          <Route path="/fahad/chat/c-ops" element={<p>ops channel page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "fahad" });
});

describe("MessageBody links", () => {
  it("opens an external URL in a new tab without handing it the opener", () => {
    renderBody("docs at https://example.com/guide.");
    const link = screen.getByRole("link", { name: "https://example.com/guide" });
    expect(link).toHaveAttribute("href", "https://example.com/guide");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link.getAttribute("rel")).toContain("noopener");
  });

  it("renders a same-origin chat URL as a pill named after the cached channel that navigates in-app", async () => {
    const url = `${window.location.origin}/fahad/chat/c-ops`;
    renderBody(`@lena join ${url}`, (client) => client.setQueryData([getChatConversationsKey, "ws-1"], [channel]));
    expect(screen.getByText("@lena")).toBeInTheDocument();
    const pill = screen.getByRole("link", { name: "#ops" });
    expect((pill as HTMLAnchorElement).href).toBe(url);
    expect(pill).toHaveAttribute("title", url);
    expect(pill).not.toHaveAttribute("target");

    await userEvent.setup().click(pill);
    expect(await screen.findByText("ops channel page")).toBeInTheDocument();
  });

  it("falls back to the kind's label when the conversation is not cached", () => {
    renderBody(`${window.location.origin}/fahad/chat/c-unknown`);
    expect(screen.getByRole("link", { name: "Chat" })).toBeInTheDocument();
  });

  it("copies a pill as its full URL rather than its label", () => {
    const url = `${window.location.origin}/fahad/chat/c-ops`;
    renderBody(`join ${url} now`, (client) => client.setQueryData([getChatConversationsKey, "ws-1"], [channel]));
    const paragraph = screen.getByText(/join/).closest("p") as HTMLParagraphElement;
    window.getSelection()?.selectAllChildren(paragraph);

    // Browsers fire copy at the selection's element; user-event fires it at the focused one, so dispatch it directly.
    const clipboard = new Map<string, string>();
    fireEvent.copy(paragraph, { clipboardData: { setData: (t: string, v: string) => clipboard.set(t, v) } });
    expect(clipboard.get("text/plain")).toBe(`join ${url} now`);
  });
});
