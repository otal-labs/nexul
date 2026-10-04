import { useEffect, useRef } from "react";

import { CollabRichTextEditor } from "@/components/doc/collab/CollabRichTextEditor";
import type { CollabSession } from "@/components/doc/collab/useCollabSession";

interface NoteFileEditorProps {
  session: CollabSession;
  markdown: string;
  conversationId: string;
}

// The note's file joined to its live room; the room seeds from the markdown and commits on the docs cadence.
export const NoteFileEditor = ({ session, markdown, conversationId }: NoteFileEditorProps) => {
  // Null until the editor reports its tree, so a commit never writes an empty file over the note.
  const bodyRef = useRef<string | null>(null);

  useEffect(() => {
    session.setGetState(() => (bodyRef.current === null ? null : { title: "", body: bodyRef.current }));
  }, [session]);

  return (
    <div className="space-y-4">
      <CollabRichTextEditor
        session={session}
        value={markdown}
        onChange={(json) => {
          bodyRef.current = json;
        }}
        aria-label="Note"
        attachTo={{ conversation_id: conversationId }}
      />
    </div>
  );
};
