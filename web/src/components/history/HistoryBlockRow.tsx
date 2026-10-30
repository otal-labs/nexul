import { HistoryText, strikeClass } from "@/components/history/HistoryText";
import { cn } from "@/lib/utils";
import type { HistoryBlockKind, HistoryRow } from "@/utils/HistoryBlocksUtility";

const MARKERS: Partial<Record<HistoryBlockKind, string>> = { bullet: "•", todo: "☐", done: "☑" };
const HEADINGS = { 1: "h1", 2: "h2", 3: "h3" } as const;
const LIST_KINDS: HistoryBlockKind[] = ["bullet", "ordered", "todo", "done"];

interface HistoryBlockRowProps {
  row: HistoryRow;
}

// One block of a version, drawn with the read-only doc's own styles (the parent carries prose-rich). A whole added or removed block
// is marked by its run's band, so its words carry no marks of their own; a removed one is struck.
export const HistoryBlockRow = ({ row }: HistoryBlockRowProps) => {
  const { block, mark, parts } = row;
  const whole = mark === "added" || mark === "removed";
  const text = (
    <>
      {!whole && <HistoryText parts={parts} />}
      {whole && <span className={cn(mark === "removed" && `${strikeClass} text-muted-foreground`)}>{block.text}</span>}
    </>
  );
  const Heading = HEADINGS[Math.min(Math.max(block.level, 1), 3) as 1 | 2 | 3];
  const marker = block.kind === "ordered" ? `${block.ordinal}.` : MARKERS[block.kind];

  return (
    <div className="[overflow-wrap:anywhere]">
      {block.kind === "heading" && <Heading>{text}</Heading>}
      {(block.kind === "paragraph" || block.kind === "other") && <p style={{ marginInlineStart: `${block.depth * 1.5}em` }}>{text}</p>}
      {block.kind === "quote" && <blockquote>{text}</blockquote>}
      {block.kind === "line" && <div className="min-h-[1lh] font-mono text-xs break-words whitespace-pre-wrap [font-variant-ligatures:none]">{text}</div>}
      {block.kind === "code" && (
        <pre>
          <code className="whitespace-pre-wrap">{text}</code>
        </pre>
      )}
      {LIST_KINDS.includes(block.kind) && (
        <div className="flex gap-2 py-0.5" style={{ paddingInlineStart: `${block.depth * 1.5}em` }}>
          <span aria-hidden className="w-5 shrink-0 text-right text-muted-foreground">
            {marker}
          </span>
          <span className="min-w-0">{text}</span>
        </div>
      )}
    </div>
  );
};
