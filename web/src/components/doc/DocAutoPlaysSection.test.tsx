import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DocAutoPlaysSection } from "@/components/doc/DocAutoPlaysSection";
import { getChatConversationsKey } from "@/hooks/ChatHooks";
import { getPlayQueueKey } from "@/hooks/PlayQueueHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { playQueue, queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(() => new Promise(() => {})) } }));

const skipped = queueItem({ target_type: "doc", target_id: "d-1", status: "skipped", reason: "no longer matches", decided_at: "2026-10-10T09:00:00Z" });

// The queue and the conversation list as already loaded, so the first render is what the reader sees.
const renderSection = (conversations: unknown[]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  client.setQueryData([getPlayQueueKey, "doc", "d-1"], playQueue({ items: [skipped] }));
  client.setQueryData([getChatConversationsKey, "ws-1"], conversations);
  render(
    <QueryClientProvider client={client}>
      <DocAutoPlaysSection workspaceId="ws-1" docId="d-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe("DocAutoPlaysSection", () => {
  it("shows a skipped line while the doc has no thread", () => {
    renderSection([]);
    expect(screen.getByText(/no longer matches/)).toHaveTextContent("Fix with AI skipped: no longer matches");
  });

  it("leaves the line to the thread once the doc has one", () => {
    renderSection([{ id: "c-1", workspace_id: "ws-1", kind: "doc_thread", doc_id: "d-1" }]);
    expect(screen.queryByText(/no longer matches/)).not.toBeInTheDocument();
  });
});
