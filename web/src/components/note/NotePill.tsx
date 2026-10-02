import { FileText } from "lucide-react";

import { formatBytes, type Attachment } from "@/models/Attachment";

interface NotePillProps {
  file: Attachment;
  onOpen: () => void;
}

// The attachment pill's shape, as a button that opens the note's file.
export const NotePill = ({ file, onOpen }: NotePillProps) => (
  <button
    type="button"
    onClick={onOpen}
    aria-haspopup="dialog"
    className="mx-1 inline-flex w-fit max-w-full items-center gap-1.5 rounded-full border border-border bg-card py-0.5 pr-2.5 pl-1 text-xs transition-colors duration-150 ease-standard hover:bg-accent/40"
  >
    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted/60">
      <FileText className="size-3 text-muted-foreground" aria-hidden />
    </span>
    <span className="truncate text-foreground">{file.name}</span>
    <span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums">{formatBytes(file.size)}</span>
  </button>
);
