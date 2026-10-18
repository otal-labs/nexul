import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useClearTicketCategory, useMoveTicketToCategory, categoryFollower } from "@/hooks/CategoryHooks";
import type { Category } from "@/models/Category";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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
  vi.mocked(toast.error).mockClear();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useMoveTicketToCategory", () => {
  it("posts the move, then refetches the categories and the board holding the ticket", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(["getTickets"], [{ id: "t-1" }]);
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useMoveTicketToCategory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ ticketId: "t-1", categoryId: "c-2" });
    expect(api.post).toHaveBeenCalledWith("/api/categories/c-2/tickets/t-1");
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getCategories"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"], exact: true });
  });
});

describe("useClearTicketCategory", () => {
  it("deletes the category link, then refetches the categories and the board holding the ticket", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(["getTickets"], [{ id: "t-1" }]);
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useClearTicketCategory(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ ticketId: "t-1" });
    expect(api.delete).toHaveBeenCalledWith("/api/categories/tickets/t-1");
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getCategories"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"], exact: true });
  });
});

describe("error paths", () => {
  it("toasts a move that the server refuses", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useMoveTicketToCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1", categoryId: "c-2" }).catch(() => {});
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
  });

  it("toasts a clear that the server refuses", async () => {
    vi.mocked(api.delete).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useClearTicketCategory(), { wrapper });
    await result.current.mutateAsync({ ticketId: "t-1" }).catch(() => {});
    await waitFor(() => expect(toast.error).toHaveBeenCalled());
  });
});

describe("the category follower", () => {
  const billing: Category = { id: "cat-1", project_id: "p-1", name: "Billing", position: 0, color: "", created_at: "", updated_at: "" };
  const lists = () =>
    seeded([
      [["getCategories"], [billing]],
      [["getProjectCategories", "p-1"], [billing]],
      [["getProjectCategories", "p-2"], []],
    ]);

  it("renames a category in both lists that hold it without a request, and refetches them when it moves", async () => {
    const client = lists();
    await followFrame(categoryFollower, "category.updated", { category: { ...billing, name: "Payments" } }, client);
    expect([client.getQueryData<Category[]>(["getCategories"])?.[0]?.name, client.getQueryData<Category[]>(["getProjectCategories", "p-1"])?.[0]?.name]).toEqual([
      "Payments",
      "Payments",
    ]);
    expect(isStale(client, ["getCategories"])).toBe(false);

    await followFrame(categoryFollower, "category.updated", { category: { ...billing, position: 3 } }, client);
    expect([["getCategories"], ["getProjectCategories", "p-1"], ["getProjectCategories", "p-2"]].map((key) => isStale(client, key))).toEqual([true, true, false]);
  });

  it("drops a deleted category from both lists", async () => {
    const client = lists();
    await followFrame(categoryFollower, "category.deleted", { category: billing }, client);
    expect([client.getQueryData(["getCategories"]), client.getQueryData(["getProjectCategories", "p-1"])]).toEqual([[], []]);
  });
});
