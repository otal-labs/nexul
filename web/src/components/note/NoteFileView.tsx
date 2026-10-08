import { FileText } from "lucide-react";

import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { useCollabSession } from "@/components/doc/collab/useCollabSession";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoteDeleteButton } from "@/components/note/NoteDeleteButton";
import { NoteFileEditor } from "@/components/note/NoteFileEditor";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useFetchMe } from "@/hooks/AuthHooks";
import { getNoteTextKey, useFetchNoteText } from "@/hooks/NoteHooks";
import { useSessionStore } from "@/stores/sessionStore";
import type { Message } from "@/models/Chat";
import { effectiveAvatar } from "@/models/User";

interface NoteFileViewProps {
  message: Message;
  fileName: string;
  canEdit: boolean;
}

// The dialog's contents: a writer joins the note's room and edits in place, a reader gets the file's render.
export const NoteFileView = ({ message, fileName, canEdit }: NoteFileViewProps) => {
  const token = useSessionStore((s) => s.token);
  const { data: me } = useFetchMe();
  const avatar = me?.user ? effectiveAvatar(me.user) : "";
  const { data: markdown, dataUpdatedAt, error, isPending } = useFetchNoteText(message.attachment_id);
  const editable = canEdit && markdown !== undefined;
  // Joins once the viewer's name is known, so presence never shows a placeholder that reconnects a moment later.
  const session = useCollabSession(editable && me ? `notes/${message.id}` : undefined, "edit", me?.user?.name ?? "", token, {
    reloadKey: [getNoteTextKey, message.attachment_id],
    ...(avatar ? { avatar } : {}),
  });

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="space-y-2 pr-8">
        <DialogDescription className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
          <FileText className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">{fileName}</span>
          {!canEdit && <span className="shrink-0 text-muted-foreground/70">· read only</span>}
        </DialogDescription>
        <DialogTitle className="text-lg leading-snug font-semibold tracking-tight">{message.body}</DialogTitle>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <DocPresenceBar participants={session?.participants ?? []} connected={session?.connected} updatedAt={message.updated_at} />
          {canEdit && <NoteDeleteButton message={message} />}
        </div>
      </div>
      <div className="rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10">
        {isPending && <LoadingDisplay label="Loading note…" />}
        {error && <ErrorDisplay error={error} title="Failed to load the note." />}
        {markdown !== undefined && session && (
          <NoteFileEditor session={session} markdown={markdown} conversationId={message.conversation_id} />
        )}
        {markdown !== undefined && !editable && <DocBodyView key={dataUpdatedAt} body={markdown} />}
      </div>
    </div>
  );
};
