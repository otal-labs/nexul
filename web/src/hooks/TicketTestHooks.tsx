import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { getTicketKey, getTicketsKey } from "@/hooks/TicketHooks";
import type { Ticket } from "@/models/Ticket";
import type { TestFailFormData, TestTarget } from "@/models/TicketTest";

export const getTestTargetKey = "getTestTarget";

export const useFetchTestTarget = (ticketId: string | undefined) =>
  useQuery({
    queryKey: [getTestTargetKey, ticketId],
    queryFn: async () => (await api.get<TestTarget>(`/api/tickets/${ticketId}/test-target`)).data,
    enabled: !!ticketId,
  });

const useInvalidateTicket = () => {
  const client = useQueryClient();
  return async (id: string) => {
    await client.invalidateQueries({ queryKey: [getTicketsKey] });
    await client.invalidateQueries({ queryKey: [getTicketKey, id] });
  };
};

export const useTestPass = () => {
  const invalidate = useInvalidateTicket();
  return useMutation({
    mutationFn: async (id: string) => (await api.post<Ticket>(`/api/tickets/${id}/test/pass`)).data,
    onSuccess: async (_, id) => {
      await invalidate(id);
      toast.success("Passed and moved to done");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useTestFail = () => {
  const invalidate = useInvalidateTicket();
  return useMutation({
    mutationFn: async ({ id, report }: { id: string; report: TestFailFormData }) =>
      (
        await api.post<Ticket>(`/api/tickets/${id}/test/fail`, {
          steps: report.steps,
          expected: report.expected,
          actual: report.actual,
          screenshots: report.screenshots.map((s) => s.id),
        })
      ).data,
    onSuccess: async (_, vars) => {
      await invalidate(vars.id);
      toast.success("Failed and sent back to progress");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
