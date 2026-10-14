import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Botwebhook } from "@/models/Botwebhook";
import { followEach, type LiveFollower } from "@/lib/live";

export const getBotwebhooksKey = "getBotwebhooks";

// The deleted list is for botwebhook:write only, so it is asked for only when the viewer holds it.
export const useFetchBotwebhooks = (conversationId: string, deleted = false, enabled = true) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  return useQuery({
    queryKey: [getBotwebhooksKey, conversationId, deleted],
    queryFn: async () =>
      (await api.get<Botwebhook[]>(`/api/conversations/${conversationId}/botwebhooks`, { params: deleted ? { deleted: true } : {} }))
        .data,
    enabled: enabled && conversationId !== "",
    meta: { workspaceId },
  });
};

const useInvalidateBots = () => {
  const client = useQueryClient();
  return (conversationId: string) => client.invalidateQueries({ queryKey: [getBotwebhooksKey, conversationId] });
};

// The caller shows a refusal (a taken name, a full conversation) inline as well.
export const useCreateBotwebhook = (conversationId: string) => {
  const invalidate = useInvalidateBots();
  return useMutation({
    mutationFn: requires("botwebhook:write", async (input: { name: string; avatar: string }) =>
      (await api.post<Botwebhook>(`/api/conversations/${conversationId}/botwebhooks`, input)).data),
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
    mutationFn: requires("botwebhook:write", async ({ bot, change }: { bot: Botwebhook; change: BotwebhookChange }) =>
      (await api.patch<Botwebhook>(`/api/botwebhooks/${bot.id}`, change)).data),
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
    mutationFn: requires("botwebhook:delete", async (bot: Botwebhook) => api.delete(`/api/botwebhooks/${bot.id}`)),
    onSuccess: async (_, bot) => {
      await invalidate(bot.conversation_id);
      toast.success("Bot deleted");
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

const refreshCreatorsBots = (client: QueryClient, userId: string, workspaceId?: string) =>
  client.invalidateQueries({
    queryKey: [getBotwebhooksKey],
    predicate: (query) => {
      if (query.queryKey[2] !== false || (workspaceId && query.meta?.workspaceId !== workspaceId)) return false;
      const bots = query.state.data as Botwebhook[] | undefined;
      return !!bots?.some((bot) => bot.created_by === userId);
    },
  });

// An open Bots section follows a bot made, changed, deleted, or restored anywhere, its URL included.
export const botwebhookFollower: LiveFollower = {
  "role.updated": ({ workspace_id }: { workspace_id: string }, { client }) =>
    client.invalidateQueries({ queryKey: [getBotwebhooksKey], predicate: (query) => query.meta?.workspaceId === workspace_id && query.queryKey[2] === false }),
  ...followEach(
    ["workspace.member.added", "workspace.member.updated", "workspace.member.removed"],
    ({ user_id, workspace_id }: { user_id: string; workspace_id: string }, { client }) => refreshCreatorsBots(client, user_id, workspace_id),
  ),
  ...followEach(
    ["account.disabled", "account.reactivated", "account.removed", "account.restored"],
    ({ account_id }: { account_id: string }, { client }) => refreshCreatorsBots(client, account_id),
  ),
  ...followEach(
    ["botwebhook.created", "botwebhook.updated", "botwebhook.deleted", "botwebhook.restored"],
    ({ conversation_id }: { conversation_id: string }, { client }) => client.invalidateQueries({ queryKey: [getBotwebhooksKey, conversation_id] }),
  ),
  // A channel rule, or the Access page's save, can stop or start a bot by what its creator may post.
  "access.grant.changed": ({ resource_type, resource_id }: { resource_type: string; resource_id: string }, { client }) => {
    if (resource_type === "channel") return client.invalidateQueries({ queryKey: [getBotwebhooksKey, resource_id] });
    if (resource_type === "workspace") return client.invalidateQueries({ queryKey: [getBotwebhooksKey] });
    return undefined;
  },
};
