import { useNavigate } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useCreateDM, useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { findDM } from "@/models/Chat";

// Opens the viewer's DM with someone, starting one only when none exists; undefined when neither is possible.
export const useOpenDirectMessage = (userId: string): (() => void) | undefined => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: me } = useFetchMe();
  const { data: conversations } = useFetchConversations(workspaceId);
  const createDM = useCreateDM(workspaceId);
  const can = useAreaAccess();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const existing = me && conversations ? findDM(conversations, [me.user.id, userId]) : undefined;
  if (!existing && !can?.("newConversation")) return undefined;

  const open = (dmId: string) => void navigate(wsPath(`/chat/${dmId}`));
  return () => {
    if (existing) {
      open(existing.id);
      return;
    }
    createDM.mutate([userId], { onSuccess: (dm) => open(dm.id) });
  };
};
