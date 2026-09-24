import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";

import { DocBodyView } from "@/components/doc/DocBodyView";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import { Input } from "@/components/ui/input";
import type { Project } from "@/models/Project";
import { reporterLabel, type Ticket } from "@/models/Ticket";
import { parseBodyToJSON } from "@/utils/RichtextUtility";

const AUTOSAVE_DEBOUNCE_MS = 800;

interface TicketDetailProps {
  ticket: Ticket;
  project?: Project;
  onSave?: (title: string, body: string) => Promise<void> | void;
}

// Mounted with key={ticket.id}: title/body seed once, so switching tickets must remount, not update in place.
export const TicketDetail = ({ ticket, project, onSave }: TicketDetailProps) => {
  const [title, setTitle] = useState(ticket.title);
  // Frozen at mount — refetched body matches what's already applied; reapplying would yank the cursor.
  const [initialBody] = useState(ticket.body);
  const [saveState, setSaveState] = useState<"idle" | "dirty" | "saving" | "saved">("idle");
  const titleRef = useRef(ticket.title);
  const bodyRef = useRef(ticket.body);
  const dirtyRef = useRef(false);
  // Matched by value, not count, so the editor's initial-content echo doesn't autosave an unchanged ticket.
  const bodySeeded = useRef(false);
  const initialBodyJSON = useRef(JSON.stringify(parseBodyToJSON(ticket.body)));
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSaveRef = useRef(onSave);
  onSaveRef.current = onSave;
  const reporter = reporterLabel(ticket.reporter);

  const flush = async () => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
    const save = onSaveRef.current;
    // An empty title never saves; the edit stays dirty until the title is restored.
    if (!save || !dirtyRef.current || titleRef.current.trim() === "") return;
    dirtyRef.current = false;
    setSaveState("saving");
    try {
      await save(titleRef.current, bodyRef.current);
      setSaveState("saved");
    } catch {
      // The mutation hook toasts the failure; dirty keeps the state honest.
      dirtyRef.current = true;
      setSaveState("dirty");
    }
  };

  const schedule = () => {
    dirtyRef.current = true;
    setSaveState("dirty");
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => void flush(), AUTOSAVE_DEBOUNCE_MS);
  };

  // A pending debounce fires directly on unmount instead of via state (no updates after unmount).
  useEffect(
    () => () => {
      if (!timer.current) return;
      clearTimeout(timer.current);
      const save = onSaveRef.current;
      if (save && titleRef.current.trim() !== "") void save(titleRef.current, bodyRef.current);
    },
    [],
  );

  return (
    <div className="space-y-6">
      <Link
        to="/board"
        className="inline-block font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
      >
        ← Board
      </Link>
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs text-muted-foreground">
            {project ? `${project.prefix}-${ticket.number}` : ticket.id}
          </span>
          <TicketStatusBadge status={ticket.status} />
        </div>
        {onSave && (
          <h1>
            <Input
              value={title}
              onChange={(e) => {
                setTitle(e.target.value);
                titleRef.current = e.target.value;
                schedule();
              }}
              onBlur={() => void flush()}
              aria-label="Ticket title"
              className="h-auto w-full rounded-none border-0 bg-transparent px-0 py-0 text-3xl font-semibold tracking-tight focus-visible:ring-0 sm:text-4xl"
            />
          </h1>
        )}
        {!onSave && <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">{ticket.title}</h1>}
        <p className="font-mono text-xs text-muted-foreground">
          created {formatUpdatedAgo(ticket.created_at)}
          {reporter && ` by ${reporter}`} · updated {formatUpdatedAgo(ticket.updated_at)}
          {saveState === "saving" && " · saving…"}
          {saveState === "saved" && " · saved"}
        </p>
      </div>
      <div className="border-t border-border pt-6" onBlur={() => void flush()}>
        {onSave && (
          <RichTextEditor
            value={initialBody}
            aria-label="Ticket description"
            attachTo={{ ticket_id: ticket.id }}
            onChange={(json) => {
              const isInitialApplication = !bodySeeded.current && json === initialBodyJSON.current;
              bodySeeded.current = true;
              bodyRef.current = json;
              if (isInitialApplication) return;
              schedule();
            }}
          />
        )}
        {!onSave && <DocBodyView body={ticket.body} />}
      </div>
    </div>
  );
};
