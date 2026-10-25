import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";

// An empty secret declines the request; the value goes to the agent's harness and is never kept by Nexul.
export const useAnswerAgentSecret = (conversationId: string) =>
  useMutation({
    mutationFn: requires("none", async ({ requestId, secret }: { requestId: string; secret: string }) =>
      api.post(`/api/agent/conversations/${conversationId}/secret`, { request_id: requestId, secret })),
    onError: (error) => toast.error(errorMessage(error)),
  });
