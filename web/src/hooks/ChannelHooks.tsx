import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { toast } from "sonner";

import type { Conversation } from "@nexul/client-core/chat";

import { api, errorMessage } from "@/api/client";
import { getChatConversationsKey, getChatUnreadKey, leaveConversationPage } from "@/hooks/ChatHooks";
import { channelMention } from "@/models/Chat";

const membersPath = (id: string) => `/api/chat/conversations/${id}/members`;

const useRefreshConversations = () => {
  const client = useQueryClient();
  return (workspaceId: string) =>
    Promise.all([
      client.invalidateQueries({ queryKey: [getChatConversationsKey, workspaceId] }),
      client.invalidateQueries({ queryKey: [getChatUnreadKey, workspaceId] }),
    ]);
};

// No error toast: going private reports through the who-stays form; going public passes its own onError.
export const useSetChannelPrivate = () => {
  const refresh = useRefreshConversations();
  return useMutation({
    mutationFn: async ({ id, isPrivate, memberIds }: { id: string; isPrivate: boolean; memberIds: string[] }) =>
      (await api.put<Conversation>(`/api/chat/conversations/${id}/private`, { private: isPrivate, member_ids: memberIds })).data,
    onSuccess: async (channel) => {
      await refresh(channel.workspace_id);
      toast.success(`${channelMention(channel)} is now ${channel.private ? "private" : "public"}`);
    },
  });
};

// Reports through the add-people form, so no error toast here.
export const useAddChannelMembers = () => {
  const refresh = useRefreshConversations();
  return useMutation({
    mutationFn: async ({ id, userIds }: { id: string; userIds: string[] }) =>
      (await api.post<Conversation>(membersPath(id), { user_ids: userIds })).data,
    onSuccess: async (channel, { userIds }) => {
      await refresh(channel.workspace_id);
      const who = userIds.length === 1 ? "1 person" : `${userIds.length} people`;
      toast.success(`Added ${who} to ${channelMention(channel)}`);
    },
  });
};

export const useRemoveChannelMember = () => {
  const refresh = useRefreshConversations();
  return useMutation({
    mutationFn: async ({ channel, userId }: { channel: Conversation; userId: string; name: string }) =>
      (await api.delete<Conversation>(`${membersPath(channel.id)}/${userId}`)).data,
    onSuccess: async (channel, { name }) => {
      await refresh(channel.workspace_id);
      toast.success(`Removed ${name} from ${channelMention(channel)}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};

// Leaving takes the channel out of the viewer's list, so a viewer of it lands on the chat home.
export const useLeaveChannel = () => {
  const refresh = useRefreshConversations();
  const navigate = useNavigate();
  const location = useLocation();
  return useMutation({
    mutationFn: async (channel: Conversation) => api.post(`/api/chat/conversations/${channel.id}/leave`),
    onSuccess: async (_result, channel) => {
      leaveConversationPage(navigate, location, channel.id);
      await refresh(channel.workspace_id);
      toast.success(`You left ${channelMention(channel)}`);
    },
    onError: (error) => toast.error(errorMessage(error)),
  });
};
