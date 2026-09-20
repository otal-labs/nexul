import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useClearTicketCategory, useMoveTicketToCategory } from "@/hooks/CategoryHooks";

vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useMoveTicketToCategory", () => {
  it("posts the ticket move", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useMoveTicketToCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1", categoryId: "c-2" });
    expect(api.post).toHaveBeenCalledWith("/api/categories/c-2/tickets/t-1");
  });

  it("invalidates the category and board keys", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useMoveTicketToCategory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ ticketId: "t-1", categoryId: "c-2" });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getCategories"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
  });
});

describe("useClearTicketCategory", () => {
  it("deletes the ticket's category link", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useClearTicketCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1" });
    expect(api.delete).toHaveBeenCalledWith("/api/categories/tickets/t-1");
  });

  it("invalidates the category and board keys", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useClearTicketCategory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ ticketId: "t-1" });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getCategories"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
  });
});

describe("error paths", () => {
  it("surfaces a move api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useMoveTicketToCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1", categoryId: "c-2" }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("surfaces a clear api error", async () => {
    vi.mocked(api.delete).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useClearTicketCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1" }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
