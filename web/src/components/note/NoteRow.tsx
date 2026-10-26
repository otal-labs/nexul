import { FileText } from "lucide-react";

import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { cn } from "@/lib/utils";
import type { Message } from "@/models/Chat";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface NoteRowProps {
  note: Message;
  fileName: string;
  selected: boolean;
  onSelect: () => void;
}

// One note in the Notes tab's rail: its file over the Agent's summary and when it was left.
export const NoteRow = ({ note, fileName, selected, onSelect }: NoteRowProps) => (
  <li>
    <button
      type="button"
      onClick={onSelect}
      aria-current={selected ? "true" : undefined}
      className={cn(
        "relative flex w-full min-w-0 flex-col gap-0.5 px-2 py-1.5 text-left transition-colors duration-150 ease-standard",
        selected ? currentRowFillClass : "hover:bg-accent/40",
      )}
    >
      {selected && <CurrentRowEdge />}
      <span className={cn("flex min-w-0 items-center gap-1.5 font-mono text-xs text-foreground", selected && "font-medium")}>
        <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <span className="truncate">{fileName}</span>
        <span className="ml-auto shrink-0 text-[11px] text-muted-foreground tabular-nums">{formatRelativeTime(note.created_at)}</span>
      </span>
      <span dir="auto" className="line-clamp-2 text-xs text-muted-foreground">
        {note.body}
      </span>
    </button>
  </li>
);
