import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react-native";
import type { ReactNode } from "react";

import { api } from "@/api/client";
import { getTicketKey, getTicketsByProjectKey, useSetTicketPerson, useUpdateTicketStatus } from "@/hooks/TicketHooks";
import { TicketRole } from "@/models/Ticket";

jest.mock("@/api/client", () => ({ api: { patch: jest.fn() } }));

const patch = jest.mocked(api.patch);

const withClient = (client: QueryClient) => {
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return Wrapper;
};

// The ticket screen caches a ticket by its id, or by its key when a chat link opened it; a change must reach both.
describe("ticket mutations", () => {
  beforeEach(() => patch.mockReset().mockResolvedValue({ id: "t-1" }));

  test.each([
    [
      "moving a ticket",
      "/api/tickets/t-1/status",
      () => {
        const { mutateAsync } = useUpdateTicketStatus();
        return () => mutateAsync({ id: "t-1", status: "st-2" });
      },
    ],
    [
      "assigning a ticket",
      "/api/tickets/t-1/developer",
      () => {
        const { mutateAsync } = useSetTicketPerson();
        return () => mutateAsync({ id: "t-1", role: TicketRole.Developer, login: "lena" });
      },
    ],
  ])("%s refreshes the board and that ticket however it was opened, and no other ticket", async (_, path, useRun) => {
    const client = new QueryClient();
    client.setQueryData([getTicketsByProjectKey, "p-1"], []);
    client.setQueryData([getTicketKey, "t-1", ""], { id: "t-1" });
    client.setQueryData([getTicketKey, "WEB-12", "acme"], { id: "t-1" });
    client.setQueryData([getTicketKey, "t-2", ""], { id: "t-2" });
    const { result } = await renderHook(useRun, { wrapper: withClient(client) });

    await act(async () => {
      await result.current();
    });

    const invalidated = (key: string[]) => client.getQueryState(key)?.isInvalidated;
    expect(patch).toHaveBeenCalledWith(path, expect.any(Object));
    expect(invalidated([getTicketsByProjectKey, "p-1"])).toBe(true);
    expect(invalidated([getTicketKey, "t-1", ""])).toBe(true);
    expect(invalidated([getTicketKey, "WEB-12", "acme"])).toBe(true);
    expect(invalidated([getTicketKey, "t-2", ""])).toBe(false);
  });
});
