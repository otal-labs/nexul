import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useFetchReviewsByTicket } from "@/hooks/ReviewHooks";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const review = {
  id: "r-1",
  pr_number: 42,
  repo: "acme/app",
  status: "approved",
  reviewer: "alice",
  created_at: "2026-08-02T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
});

describe("useFetchReviewsByTicket", () => {
  it("is disabled without a ticket id", () => {
    const { result } = renderHook(() => useFetchReviewsByTicket(undefined), { wrapper });
    expect(result.current.isPending).toBe(true);
    expect(api.get).not.toHaveBeenCalled();
  });

  it("loads reviews for a ticket", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [review] });
    const { result } = renderHook(() => useFetchReviewsByTicket("t-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([review]));
    expect(api.get).toHaveBeenCalledWith("/api/reviews", { params: { ticket_id: "t-1" } });
  });
});
