import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocThreadButton } from "@/components/chat/DocThreadButton";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (err: unknown) => (err instanceof Error ? err.message : String(err)),
}));

const conversation = {
  id: "thread-1",
  workspace_id: "ws-1",
  kind: "doc_thread",
  doc_id: "doc-1",
  created_by: "u1",
  created_at: "",
  updated_at: "",
};

const renderButton = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<DocThreadButton workspaceId="ws-1" docId="doc-1" />} />
          <Route path="/chat/:conversationId" element={<div>chat-page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocThreadButton", () => {
  it("is hidden for a caller without docs:thread", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { role_name: "member", permissions: [] } });
    renderButton();
    // Let the role query settle before asserting absence, or the button could just not have rendered yet.
    await new Promise((r) => setTimeout(r, 0));
    expect(screen.queryByRole("button", { name: "Thread" })).not.toBeInTheDocument();
  });

  it("gets or creates the thread and navigates to it for a caller with docs:thread", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockResolvedValue({ data: { role_name: "member", permissions: ["docs:thread"] } });
    vi.mocked(api.post).mockResolvedValue({ data: conversation });
    renderButton();

    const button = await screen.findByRole("button", { name: "Thread" });
    await user.click(button);

    expect(api.post).toHaveBeenCalledWith("/api/chat/docs/doc-1/thread", { workspace_id: "ws-1" });
    expect(await screen.findByText("chat-page")).toBeInTheDocument();
  });
});
