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
  const listed = (over: object) => ({ id: "t-1", project_id: "p-1", title: "Fix", body: "", status: "st-1", updated_at: "2026-11-25T10:00:00Z", ...over });
  const answer = { ...listed({ status: "st-2", developer: "lena", updated_at: "2026-11-25T10:01:00Z" }), body: "full body" };
  beforeEach(() => patch.mockReset().mockResolvedValue(answer));

  test.each([
    [
      "moving a ticket",
      "/api/tickets/t-1/status",
      [15_000],
      () => {
        const { mutateAsync } = useUpdateTicketStatus();
        return () => mutateAsync({ id: "t-1", status: "st-2" });
      },
    ],
    [
      "assigning a ticket",
      "/api/tickets/t-1/developer",
      [],
      () => {
        const { mutateAsync } = useSetTicketPerson();
        return () => mutateAsync({ id: "t-1", role: TicketRole.Developer, login: "lena" });
      },
    ],
  ] as const)("%s writes the server's copy into the board and that ticket however it was opened, refetching nothing", async (_, path, deadline, useRun) => {
    const client = new QueryClient();
    const other = listed({ id: "t-2" });
    client.setQueryData([getTicketsByProjectKey, "p-1"], [listed({}), other]);
    client.setQueryData([getTicketKey, "t-1", ""], listed({ body: "old body" }));
    client.setQueryData([getTicketKey, "WEB-12", "acme"], listed({ body: "old body" }));
    client.setQueryData([getTicketKey, "t-2", ""], other);
    const invalidate = jest.spyOn(client, "invalidateQueries");
    const { result } = await renderHook(useRun, { wrapper: withClient(client) });

    await act(async () => {
      await result.current();
    });

    expect(patch).toHaveBeenCalledWith(path, expect.any(Object), ...deadline);
    expect(client.getQueryData([getTicketsByProjectKey, "p-1"])).toEqual([{ ...answer, body: "" }, other]);
    expect(client.getQueryData([getTicketKey, "t-1", ""])).toEqual(answer);
    expect(client.getQueryData([getTicketKey, "WEB-12", "acme"])).toEqual(answer);
    expect(client.getQueryData([getTicketKey, "t-2", ""])).toBe(other);
    expect(invalidate).not.toHaveBeenCalled();
  });
});
