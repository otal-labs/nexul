import { FileText } from "lucide-react";

import { previewLines, type NoteVariant } from "@/components/note/noteVariants";
import type { Attachment } from "@/models/Attachment";
import { formatBytes } from "@/models/Attachment";

interface NotePillProps {
  pill: NoteVariant["pill"];
  summary: string;
  file: Attachment | undefined;
  text: string | undefined;
  onOpen: () => void;
}

const fileName = (file: Attachment | undefined) => file?.name ?? "note.md";

// Prototype only: three ways a note reads in the thread, all opening the same dialog.
export const NotePill = ({ pill, summary, file, text, onOpen }: NotePillProps) => (
  <div className="space-y-2 px-1 py-1">
    {pill !== "title" && <p className="text-sm">{summary}</p>}
    {pill === "file" && (
      <button
        type="button"
        onClick={onOpen}
        className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-border bg-card py-0.5 pr-2.5 pl-1 text-xs transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted/60">
          <FileText className="size-3 text-muted-foreground" aria-hidden />
        </span>
        <span className="truncate text-foreground">{fileName(file)}</span>
        {file && <span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums">{formatBytes(file.size)}</span>}
      </button>
    )}
    {pill === "preview" && (
      <button
        type="button"
        onClick={onOpen}
        className="block w-full rounded-lg border border-border bg-card px-3 py-2 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <span className="flex items-center gap-1.5 font-mono text-[11px] text-muted-foreground">
          <FileText className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">{fileName(file)}</span>
        </span>
        <span className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground [mask-image:linear-gradient(to_bottom,black_45%,transparent)]">
          {previewLines(text ?? "").join(" ")}
        </span>
      </button>
    )}
    {pill === "title" && (
      <button type="button" onClick={onOpen} className="group/note text-left">
        <span className="text-sm font-medium underline decoration-border underline-offset-4 transition-colors duration-150 ease-standard group-hover/note:decoration-foreground">
          {summary}
        </span>
        <span className="ml-1.5 inline-flex items-center gap-1 rounded-md border border-border px-1.5 py-px align-[1px] font-mono text-[10px] text-muted-foreground">
          <FileText className="size-3" aria-hidden />
          md
        </span>
      </button>
    )}
  </div>
);
