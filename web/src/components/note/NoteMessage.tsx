import { useState } from "react";

import { NoteDialog } from "@/components/note/NoteDialog";
import { NoteDocument } from "@/components/note/NoteDocument";
import { NotePill } from "@/components/note/NotePill";
import { useNoteVariant } from "@/components/note/noteVariants";
import { useAttachmentText, useFetchAttachments } from "@/hooks/AttachmentHooks";
import type { Message } from "@/models/Chat";

interface NoteMessageProps {
  message: Message;
}

// Prototype only: an Agent message carrying a note's file, read through the existing attachment routes.
export const NoteMessage = ({ message }: NoteMessageProps) => {
  const [open, setOpen] = useState(false);
  const { variant, reader } = useNoteVariant();
  const { data: files } = useFetchAttachments({ conversation_id: message.conversation_id });
  const { data: text } = useAttachmentText(message.attachment_id);
  const file = files?.find((f) => f.id === message.attachment_id);

  return (
    <>
      <NotePill pill={variant.pill} summary={message.body} file={file} text={text} onOpen={() => setOpen(true)} />
      <NoteDialog kind={variant.dialog} open={open} onOpenChange={setOpen} title={message.body}>
        {text !== undefined && (
          <NoteDocument
            key={`${variant.key}-${reader}`}
            variant={variant}
            reader={reader}
            summary={message.body}
            file={file}
            text={text}
            conversationId={message.conversation_id}
            fill={variant.dialog === "tall"}
          />
        )}
      </NoteDialog>
    </>
  );
};
