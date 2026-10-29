import { HistoryText } from "@/components/history/HistoryText";
import { cn } from "@/lib/utils";
import type { HistoryBlockKind, HistoryRow, RowMark } from "@/utils/HistoryBlocksUtility";

// A short bar in the left margin of a changed block: grey for edited, green for added, red for removed.
const BAR: Record<RowMark, string> = {
  same: "",
  edited: "before:bg-muted-foreground",
  added: "before:bg-success",
  removed: "before:bg-destructive",
};
const BAR_SHAPE = "before:absolute before:inset-y-1 before:-left-5 before:w-[3px] before:rounded-sm";

const MARKERS: Partial<Record<HistoryBlockKind, string>> = { bullet: "•", todo: "☐", done: "☑" };
const HEADINGS = { 1: "h1", 2: "h2", 3: "h3" } as const;
const LIST_KINDS: HistoryBlockKind[] = ["bullet", "ordered", "todo", "done"];

interface HistoryBlockRowProps {
  row: HistoryRow;
}

// One block of a version, drawn with the read-only doc's own styles (the parent carries prose-rich).
export const HistoryBlockRow = ({ row }: HistoryBlockRowProps) => {
  const { block, mark, parts } = row;
  const text = <HistoryText parts={parts} />;
  const Heading = HEADINGS[Math.min(Math.max(block.level, 1), 3) as 1 | 2 | 3];
  const marker = block.kind === "ordered" ? `${block.ordinal}.` : MARKERS[block.kind];

  return (
    <div className={cn("relative", mark !== "same" && BAR_SHAPE, BAR[mark])}>
      {block.kind === "heading" && <Heading>{text}</Heading>}
      {(block.kind === "paragraph" || block.kind === "other") && <p style={{ marginInlineStart: `${block.depth * 1.5}em` }}>{text}</p>}
      {block.kind === "quote" && <blockquote>{text}</blockquote>}
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
