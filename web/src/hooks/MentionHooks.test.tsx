import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { refsKey, resolveMentions, searchMentions, useResolveMentions } from "@/hooks/MentionHooks";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

describe("MentionHooks", () => {
  it("refsKey is order-independent so all chips share one batch query", () => {
    expect(refsKey([{ type: "ticket", id: "t-1" }, { type: "doc", id: "d-9" }])).toBe(
      refsKey([{ type: "doc", id: "d-9" }, { type: "ticket", id: "t-1" }]),
    );
  });

  it("resolveMentions posts the full ref set in one request", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips: [{ type: "ticket", id: "t-1", title: "x", can_open: true }] } });
    const chips = await resolveMentions([{ type: "ticket", id: "t-1" }]);
    expect(api.post).toHaveBeenCalledWith("/api/mentions/resolve", { refs: [{ type: "ticket", id: "t-1" }] });
    expect(chips).toHaveLength(1);
  });

  it("searchMentions queries the picker source", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { results: [] } });
    await searchMentions("fix");
    expect(api.get).toHaveBeenCalledWith("/api/mentions/search", { params: { q: "fix", limit: 8 } });
  });

  it("useResolveMentions is disabled for an empty ref set", async () => {
    const { result } = renderHook(() => useResolveMentions([]), { wrapper });
    expect(result.current.isFetching).toBe(false);
    expect(api.post).not.toHaveBeenCalled();
  });

  it("useResolveMentions resolves a non-empty ref set", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { chips: [] } });
    const { result } = renderHook(() => useResolveMentions([{ type: "doc", id: "d-9" }]), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.post).toHaveBeenCalledTimes(1);
  });
});
