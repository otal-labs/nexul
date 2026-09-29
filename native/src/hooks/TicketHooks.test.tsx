import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { getTicketKey, getTicketsByProjectKey, useUpdateTicketStatus } from "@/hooks/TicketHooks";

jest.mock("@/api/client", () => ({ api: { patch: jest.fn() } }));

const patch = jest.mocked(api.patch);

const withClient = (client: QueryClient) => {
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return Wrapper;
};

describe("useUpdateTicketStatus", () => {
  beforeEach(() => patch.mockReset());

  test("moving a ticket invalidates the board list and the ticket detail", async () => {
    patch.mockResolvedValue({ id: "t-1", status: "st-2" });
    const client = new QueryClient();
    const invalidate = jest.spyOn(client, "invalidateQueries");
    const { result } = await renderHook(() => useUpdateTicketStatus(), { wrapper: withClient(client) });

    result.current.mutate({ id: "t-1", status: "st-2" });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "st-2" });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getTicketsByProjectKey] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [getTicketKey, "t-1"] });
  });
});
