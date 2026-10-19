import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ConversationThread } from "@/components/chat/ConversationThread";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (err: unknown) => (err instanceof Error ? err.message : String(err)),
}));

const responses: Record<string, unknown> = {
  "/api/docs/doc-1": { id: "doc-1", project_id: "p-1", title: "Phase 1 feature list" },
  "/api/tickets/t-1": { id: "t-1", project_id: "p-1", number: 7, title: "Fix login" },
  "/api/projects": [{ id: "p-1", prefix: "WEB", name: "Web" }],
  "/api/auth/me": { user: { id: "u1" } },
};

const renderThread = (conversation: Record<string, unknown>) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ConversationThread
          workspaceId="ws-1"
          conversation={{ id: "c-1", workspace_id: "ws-1", created_by: "u1", created_at: "", updated_at: "", ...conversation } as never}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation((url) => Promise.resolve({ data: responses[url] ?? [] }));
});

describe("ConversationThread header", () => {
  it("links a doc thread back to its doc", async () => {
    renderThread({ kind: "doc_thread", doc_id: "doc-1" });

    expect(await screen.findByRole("link", { name: "Phase 1 feature list" })).toHaveAttribute("href", "/acme/docs/WEB/doc-1");
  });

  it("links a ticket thread back to its ticket by key", async () => {
    renderThread({ kind: "ticket_thread", ticket_id: "t-1" });

    expect(await screen.findByRole("link", { name: "Fix login" })).toHaveAttribute("href", "/acme/tickets/WEB-7");
  });
});
