import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import { ticketChanged } from "@/hooks/TicketCache";
import type { Ticket } from "@/models/Ticket";
import type { TestFailFormData, TestTarget } from "@/models/TicketTest";

export const getTestTargetKey = "getTestTarget";

export const useFetchTestTarget = (ticketId: string | undefined) =>
  useQuery({
    queryKey: [getTestTargetKey, ticketId],
    queryFn: async () => (await api.get<TestTarget>(`/api/tickets/${ticketId}/test-target`)).data,
    enabled: !!ticketId,
  });

export const useTestPass = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => (await api.post<Ticket>(`/api/tickets/${id}/test/pass`)).data,
    onSuccess: async (ticket) => {
      await ticketChanged(client, ticket);
      toast.success("Passed and moved to done");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useTestFail = () => {
  const client = useQueryClient();
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
    onSuccess: async (ticket) => {
      await ticketChanged(client, ticket);
      toast.success("Failed and sent back to progress");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
