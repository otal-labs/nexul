import { MessageSquare } from "lucide-react";
import { useNavigate } from "react-router";

import { ActionButton } from "@/components/action/ActionButton";
import { useGetOrCreateDocThread } from "@/hooks/ChatHooks";
import { useEntityActions } from "@/hooks/useEntityActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface DocThreadButtonProps {
  workspaceId: string;
  docId: string;
  projectId: string;
}

// Opens the doc's one thread on the chat page (ADR 0093), created on first use like a ticket's.
export const DocThreadButton = ({ workspaceId, docId, projectId }: DocThreadButtonProps) => {
  const answer = useEntityActions("doc", docId);
  const getOrCreate = useGetOrCreateDocThread(workspaceId);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();

  return (
    <ActionButton
      permission="docs:thread"
      projectId={projectId}
      answer={answer("docs:thread")}
      variant="ghost"
      size="icon"
      className="size-7"
      aria-label="Thread"
      icon={MessageSquare}
      onClick={() => getOrCreate.mutate(docId, { onSuccess: (conversation) => void navigate(wsPath(`/chat/${conversation.id}`)) })}
    />
  );
};
