import { useNavigate } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useCreateDM, useFetchConversations } from "@/hooks/ChatHooks";
import { usePerson } from "@/hooks/PeopleHooks";
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
  const owner = usePerson(userId).owner === true;
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const existing = me && conversations ? findDM(conversations, [me.user.id, userId]) : undefined;
  // The server lets anyone message an Owner, so only a DM with someone else takes chat:write.
  if (!existing && !owner && !can?.("newConversation")) return undefined;

  const open = (dmId: string) => void navigate(wsPath(`/chat/${dmId}`));
  return () => {
    if (existing) {
      open(existing.id);
      return;
    }
    createDM.mutate([userId], { onSuccess: (dm) => open(dm.id) });
  };
};
