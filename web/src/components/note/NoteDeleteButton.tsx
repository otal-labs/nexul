import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useDeleteMessage } from "@/hooks/ChatHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Message } from "@/models/Chat";

interface NoteDeleteButtonProps {
  message: Message;
}

// Deleting the note's message takes its file and the images it shows with it; the thread row then drops the dialog.
export const NoteDeleteButton = ({ message }: NoteDeleteButtonProps) => {
  const { open: confirmDelete } = useConfirmationDialog();
  const deleteMessage = useDeleteMessage(message.conversation_id);

  const onDelete = async () => {
    const ok = await confirmDelete({
      title: "Delete note",
      message: "Delete this note? Its file and images go with it.",
      confirmLabel: "Delete note",
    });
    if (ok) deleteMessage.mutate(message.id);
  };

  return (
    <Button variant="ghost" size="sm" loading={deleteMessage.isPending} onClick={() => void onDelete()}>
      {!deleteMessage.isPending && <Trash2 aria-hidden />}
      Delete
    </Button>
  );
};
