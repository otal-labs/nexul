import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { CodeReview } from "@/models/CodeReview";

const getReviewsByTicketKey = "getReviewsByTicket";

export const useFetchReviewsByTicket = (ticketId: string | undefined) =>
  useQuery({
    queryKey: [getReviewsByTicketKey, ticketId],
    queryFn: async () => (await api.get<CodeReview[]>("/api/reviews", { params: { ticket_id: ticketId } })).data,
    enabled: !!ticketId,
  });
