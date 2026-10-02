import { AttachmentRow } from "@/components/attachment/AttachmentRow";
import { useFetchAttachments } from "@/hooks/AttachmentHooks";

interface NoteFilePillProps {
  conversationId: string;
  attachmentId: string;
}

// A note's file as the plain attachment pill, found among its thread's files.
export const NoteFilePill = ({ conversationId, attachmentId }: NoteFilePillProps) => {
  const { data: files } = useFetchAttachments({ conversation_id: conversationId });
  const file = files?.find((f) => f.id === attachmentId);
  if (!file) return null;
  return (
    <ul className="mt-1.5">
      <AttachmentRow attachment={file} canDelete={false} />
    </ul>
  );
};
