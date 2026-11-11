import { useEffect } from "react";

import { useProductName } from "@/ee/hooks/useProductName";
import { useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { groupConversations } from "@/models/Chat";

// The tab title counts unread direct messages, so a waiting DM shows while another tab is in front.
export const useUnreadTitle = (enabled: boolean) => {
  const name = useProductName();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(enabled ? workspaceId : undefined);
  const { data: unread } = useFetchChatUnread(workspaceId, enabled);
  const count = groupConversations(conversations ?? []).dms.reduce((sum, dm) => sum + (unread?.[dm.id] ?? 0), 0);
  useEffect(() => {
    document.title = count > 0 ? `(${count}) ${name}` : name;
  }, [count, name]);
};
