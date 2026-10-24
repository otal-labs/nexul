import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { getChatUnreadKey } from "@/hooks/ChatHooks";
import { useUnreadTitle } from "@/hooks/useUnreadTitle";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

const conversations = [
  { id: "dm-1", kind: "dm" },
  { id: "dm-2", kind: "dm" },
  { id: "ch-1", kind: "channel" },
  { id: "th-1", kind: "doc_thread" },
];

const setup = (unread: Record<string, number>) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => ({ data: url === "/api/chat/unread" ? unread : conversations }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  return { client, wrapper };
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  document.title = "Nexul";
});

describe("useUnreadTitle", () => {
  it("counts unread direct messages only, and drops the count once they are read", async () => {
    const { client, wrapper } = setup({ "dm-1": 2, "dm-2": 1, "ch-1": 5, "th-1": 4 });
    renderHook(() => useUnreadTitle(true), { wrapper });
    await waitFor(() => expect(document.title).toBe("(3) Nexul"));

    act(() => client.setQueryData([getChatUnreadKey, "ws-1"], { "ch-1": 5 }));
    await waitFor(() => expect(document.title).toBe("Nexul"));
  });

  it("asks for nothing and keeps the plain title while signed out", () => {
    const { wrapper } = setup({ "dm-1": 2 });
    renderHook(() => useUnreadTitle(false), { wrapper });
    expect(api.get).not.toHaveBeenCalled();
    expect(document.title).toBe("Nexul");
  });
});
