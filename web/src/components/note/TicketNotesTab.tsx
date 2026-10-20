import { useId } from "react";
import { useSearchParams } from "react-router";

import { microheaderClass } from "@/components/Microheader";
import { NoteFileView } from "@/components/note/NoteFileView";
import { NoteRow } from "@/components/note/NoteRow";
import { useFetchAttachments } from "@/hooks/AttachmentHooks";
import { useCanEditNote } from "@/hooks/NoteHooks";
import type { TicketNotes } from "@/models/Chat";

interface TicketNotesTabProps {
  ticketId: string;
  notes: TicketNotes;
  gridClassName: string;
  railClassName: string;
}

const fallbackName = "note.md";

// The ticket's Notes tab, laid out as its Ticket tab: the picked note open on the left, every note in the rail.
export const TicketNotesTab = ({ ticketId, notes, gridClassName, railClassName }: TicketNotesTabProps) => {
  const headingId = useId();
  const [params, setParams] = useSearchParams();
  const canEdit = useCanEditNote(ticketId);
  const { data: files } = useFetchAttachments({ conversation_id: notes.conversation_id });
  const nameOf = (attachmentId: string | undefined) => files?.find((f) => f.id === attachmentId)?.name ?? fallbackName;
  const selected = notes.notes.find((n) => n.id === params.get("note")) ?? notes.notes[0];
  const select = (id: string) =>
    setParams(
      (next) => {
        next.set("note", id);
        return next;
      },
      { replace: true },
    );

  return (
    <div className={gridClassName}>
      {selected && <NoteFileView key={selected.id} message={selected} fileName={nameOf(selected.attachment_id)} canEdit={canEdit} inline />}
      <section className={railClassName} aria-labelledby={headingId}>
        <h2 id={headingId} className={`${microheaderClass} mb-2 px-2`}>
          Notes
        </h2>
        <ul className="space-y-0.5">
          {notes.notes.map((note) => (
            <NoteRow
              key={note.id}
              note={note}
              fileName={nameOf(note.attachment_id)}
              selected={note.id === selected?.id}
              onSelect={() => select(note.id)}
            />
          ))}
        </ul>
      </section>
    </div>
  );
};
