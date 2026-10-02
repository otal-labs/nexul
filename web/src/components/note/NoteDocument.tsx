import { FileText } from "lucide-react";
import { useState } from "react";

import { colorFor, type CollabParticipant } from "@/components/doc/collab/useCollabSession";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocPresenceBar } from "@/components/doc/DocPresenceBar";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import type { NoteVariant } from "@/components/note/noteVariants";
import { Switch } from "@/components/ui/switch";
import { useFetchMe } from "@/hooks/AuthHooks";
import { cn } from "@/lib/utils";
import type { Attachment } from "@/models/Attachment";

interface NoteDocumentProps {
  variant: NoteVariant;
  reader: boolean;
  summary: string;
  file: Attachment | undefined;
  text: string;
  conversationId: string;
  /** Lets the tall dialog's card fill the height and scroll inside. */
  fill?: boolean;
}

// Prototype only: the note's file framed like the ticket body card, with a stand-in presence bar (no live session yet).
export const NoteDocument = ({ variant, reader, summary, file, text, conversationId, fill = false }: NoteDocumentProps) => {
  const { data: me } = useFetchMe();
  const [switchedOn, setSwitchedOn] = useState(false);
  const [draft, setDraft] = useState(text);
  const editing = !reader && (variant.editing === "always" || switchedOn);
  const participants: CollabParticipant[] = [
    { clientID: 1, name: me?.user.name ?? "You", color: colorFor(1), activity: editing ? "editing" : "viewing" },
    { clientID: 2, name: "Sam Rivera", color: colorFor(2), activity: "editing" },
  ];

  return (
    <div className={cn("flex min-h-0 flex-col gap-4", fill && "h-full")}>
      <div className="space-y-2 pr-8">
        <p className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
          <FileText className="size-3.5" aria-hidden />
          {file?.name ?? "note.md"}
          {reader && <span className="text-muted-foreground/70">· read only</span>}
        </p>
        <h2 className="text-lg font-semibold tracking-tight">{summary}</h2>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <DocPresenceBar participants={participants} connected updatedAt={file?.created_at ?? new Date().toISOString()} />
          {!reader && variant.editing === "switch" && (
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Edit
              <Switch checked={switchedOn} onCheckedChange={setSwitchedOn} aria-label="Edit note" />
            </label>
          )}
        </div>
      </div>
      <div
        className={cn(
          "rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10",
          fill && "min-h-0 flex-1 overflow-y-auto",
          editing && "ring-1 ring-ring/20",
        )}
      >
        {editing && (
          <RichTextEditor
            value={draft}
            aria-label="Note"
            attachTo={{ conversation_id: conversationId }}
            onChange={setDraft}
          />
        )}
        {!editing && <DocBodyView body={draft} />}
      </div>
    </div>
  );
};
