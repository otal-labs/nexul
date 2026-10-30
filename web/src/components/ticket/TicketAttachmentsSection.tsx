import { Link } from "react-router";

import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { useTicketThreadMessages } from "@/hooks/ChatHooks";
import { useTabPath } from "@/hooks/useTabPath";

interface TicketAttachmentsSectionProps {
  ticketId: string;
  projectId: string;
  workspaceId: string;
}

// The ticket's own files, with a line to the files posted in its thread, which are held there and not here.
export const TicketAttachmentsSection = ({ ticketId, projectId, workspaceId }: TicketAttachmentsSectionProps) => {
  const { data: messages } = useTicketThreadMessages(workspaceId, ticketId);
  const { tabPath } = useTabPath();
  const files = (messages ?? []).filter((m) => !m.deleted_at && m.attachment_id).length;

  return (
    <AttachmentsSection
      owner={{ ticket_id: ticketId }}
      projectId={projectId}
      className="px-2 [&_[role=status]]:text-xs"
      actionPlacement="end"
      aside={
        files > 0 && (
          <Link to={tabPath("thread")} className="block text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
            {files === 1 ? "1 file in the thread" : `${files} files in the thread`}
          </Link>
        )
      }
    />
  );
};
