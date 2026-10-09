import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useFetchChatThreadIndicators, useFetchChatUnread, useFetchConversations, useFetchMessages, usePostMessage } from "@/hooks/ChatHooks";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: (e: unknown) => String(e),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("useFetchConversations", () => {
  it("loads conversations for the workspace, skipping without one", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ id: "c1" }] });
    const { result } = renderHook(() => useFetchConversations("ws-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([{ id: "c1" }]));
    expect(api.get).toHaveBeenCalledWith("/api/chat/conversations", { params: { workspace_id: "ws-1" } });

    vi.mocked(api.get).mockClear();
    const { result: idle } = renderHook(() => useFetchConversations(undefined), { wrapper });
    expect(idle.current.fetchStatus).toBe("idle");
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useFetchChatThreadIndicators", () => {
  // Naming every ticket put ~37 bytes per card in the URL, past nginx's header buffer on a big board.
  it("asks once per project, naming the project rather than every ticket on the board", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { "t-1": true } });
    const { result } = renderHook(() => useFetchChatThreadIndicators("p-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ "t-1": true }));
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.get).toHaveBeenCalledWith("/api/chat/tickets/thread-status", { params: { project_id: "p-1" } });
  });
});

describe("useFetchMessages", () => {
  it("loads a conversation's messages", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ id: "m1" }] });
    const { result } = renderHook(() => useFetchMessages("c1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([{ id: "m1" }]));
    expect(api.get).toHaveBeenCalledWith("/api/chat/conversations/c1/messages", { params: undefined });
  });
});

describe("useFetchChatUnread", () => {
  it("loads unread counts per conversation", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { c1: 3 } });
    const { result } = renderHook(() => useFetchChatUnread("ws-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ c1: 3 }));
    expect(api.get).toHaveBeenCalledWith("/api/chat/unread", { params: { workspace_id: "ws-1" } });
  });
});

describe("usePostMessage", () => {
  it("shows the message as a pending row on send and swaps in the server copy on success", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapperWith = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    client.setQueryData(["getMe"], { user: { id: "u1" } });
    client.setQueryData(["getChatMessages", "c1", undefined], []);
    const saved = { id: "m1", conversation_id: "c1", author_id: "u1", author_kind: "user", body: "hello", mentions: null };
    let resolvePost: (value: { data: typeof saved }) => void = () => {};
    vi.mocked(api.post).mockReturnValue(new Promise((resolve) => (resolvePost = resolve)));

    const { result } = renderHook(() => usePostMessage("c1"), { wrapper: wrapperWith });
    act(() => result.current.mutate("hello"));
    await waitFor(() => expect(client.getQueryData(["getChatMessages", "c1", undefined])).toHaveLength(1));
    const [pending] = client.getQueryData<{ id: string }[]>(["getChatMessages", "c1", undefined])!;
    expect(pending).toMatchObject({ body: "hello", author_id: "u1", pending: true });

    act(() => resolvePost({ data: saved }));
    // The server copy keeps the pending row's key, so the row on screen is the same one and its entrance isn't cut short.
    await waitFor(() => expect(client.getQueryData(["getChatMessages", "c1", undefined])).toEqual([{ ...saved, client_key: pending!.id }]));
  });

  it("drops the pending row when the post fails", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapperWith = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    client.setQueryData(["getChatMessages", "c1", undefined], []);
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));

    const { result } = renderHook(() => usePostMessage("c1"), { wrapper: wrapperWith });
    act(() => result.current.mutate("hello"));
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(client.getQueryData(["getChatMessages", "c1", undefined])).toEqual([]);
  });
});
