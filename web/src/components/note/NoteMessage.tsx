import { useState } from "react";

import { NoteFileView } from "@/components/note/NoteFileView";
import { NotePill } from "@/components/note/NotePill";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useFetchAttachments } from "@/hooks/AttachmentHooks";
import { useCanEditNote } from "@/hooks/NoteHooks";
import type { Message } from "@/models/Chat";

interface NoteMessageProps {
  message: Message;
  ticketId: string | undefined;
}

// A note's file pill under the Agent's summary, opening the file in a dialog the width of the ticket body card.
export const NoteMessage = ({ message, ticketId }: NoteMessageProps) => {
  const [open, setOpen] = useState(false);
  const canEdit = useCanEditNote(ticketId);
  const { data: files } = useFetchAttachments({ conversation_id: message.conversation_id });
  const file = files?.find((f) => f.id === message.attachment_id);

  return (
    <>
      {file && <NotePill file={file} onOpen={() => setOpen(true)} />}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[calc(100dvh-4rem)] overflow-y-auto bg-popover sm:max-w-[min(48rem,calc(100%-2rem))]">
          <NoteFileView message={message} fileName={file?.name ?? "note.md"} canEdit={canEdit} />
        </DialogContent>
      </Dialog>
    </>
  );
};
