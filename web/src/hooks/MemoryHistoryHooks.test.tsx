import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { getHistoryKey } from "@/hooks/HistoryHooks";
import { useRestoreMemoryVersion } from "@/hooks/MemoryHistoryHooks";
import { getMemoryKey, memoryHistoryUrl } from "@/hooks/MemoryHooks";

vi.mock("@/api/client", () => ({ api: { post: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

describe("useRestoreMemoryVersion", () => {
  it("posts the restore for the version and reads the memory and its history again", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { id: "mem-1" } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useRestoreMemoryVersion("mem-1"), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });

    await result.current.mutateAsync(1);

    expect(api.post).toHaveBeenCalledWith("/api/memories/mem-1/versions/1/restore");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getMemoryKey, "mem-1"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getHistoryKey, memoryHistoryUrl("mem-1")] });
  });
});
