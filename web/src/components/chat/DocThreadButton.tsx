import { MessageSquare } from "lucide-react";
import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { useGetOrCreateDocThread } from "@/hooks/ChatHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";

interface DocThreadButtonProps {
  workspaceId: string;
  docId: string;
}

// Opens the doc's one thread on the chat page (ADR 0060), created on first use like a ticket's; hidden
// for a caller without docs:thread on this doc.
export const DocThreadButton = ({ workspaceId, docId }: DocThreadButtonProps) => {
  const canThread = useHasPermission("docs:thread");
  const getOrCreate = useGetOrCreateDocThread(workspaceId);
  const navigate = useNavigate();

  if (!canThread) return null;

  return (
    <Button
      variant="ghost"
      size="icon"
      className="size-7"
      aria-label="Thread"
      onClick={() => getOrCreate.mutate(docId, { onSuccess: (conversation) => void navigate(`/chat/${conversation.id}`) })}
    >
      <MessageSquare className="size-4" aria-hidden />
    </Button>
  );
};
