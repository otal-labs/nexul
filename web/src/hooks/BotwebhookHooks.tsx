import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, errorMessage } from "@/api/client";
import type { Botwebhook } from "@/models/Botwebhook";

export const getBotwebhooksKey = "getBotwebhooks";

// The deleted list is for botwebhook:write only, so it is asked for only when the viewer holds it.
export const useFetchBotwebhooks = (conversationId: string, deleted = false, enabled = true) =>
  useQuery({
    queryKey: [getBotwebhooksKey, conversationId, deleted],
    queryFn: async () =>
      (await api.get<Botwebhook[]>(`/api/conversations/${conversationId}/botwebhooks`, { params: deleted ? { deleted: true } : {} }))
        .data,
    enabled: enabled && conversationId !== "",
  });

const useInvalidateBots = () => {
  const client = useQueryClient();
  return (conversationId: string) => client.invalidateQueries({ queryKey: [getBotwebhooksKey, conversationId] });
};

// The caller shows a refusal (a taken name, a full conversation) inline as well.
export const useCreateBotwebhook = (conversationId: string) => {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: async (input: { name: string; avatar: string }) =>
      (await api.post<Botwebhook>(`/api/conversations/${conversationId}/botwebhooks`, input)).data,
    onSuccess: async () => {
      await invalidate(conversationId);
      toast.success("Bot created");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export type BotwebhookChange = { name: string } | { avatar: string } | { regenerate: true } | { deleted: false };

const changeToast = (change: BotwebhookChange): string => {
  if ("name" in change) return "Bot renamed";
  if ("avatar" in change) return "Avatar changed";
  if ("regenerate" in change) return "URL regenerated";
  return "Bot restored";
};

export const useUpdateBotwebhook = () => {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: async ({ bot, change }: { bot: Botwebhook; change: BotwebhookChange }) =>
      (await api.patch<Botwebhook>(`/api/botwebhooks/${bot.id}`, change)).data,
    onSuccess: async (_, { bot, change }) => {
      await invalidate(bot.conversation_id);
      toast.success(changeToast(change));
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

export const useDeleteBotwebhook = () => {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: async (bot: Botwebhook) => api.delete(`/api/botwebhooks/${bot.id}`),
    onSuccess: async (_, bot) => {
      await invalidate(bot.conversation_id);
      toast.success("Bot deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
