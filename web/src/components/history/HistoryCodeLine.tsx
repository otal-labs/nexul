import { cn } from "@/lib/utils";
import type { CodeLine, CodeSign } from "@/utils/HistoryCodeUtility";

const ROW: Record<CodeSign, string> = {
  context: "",
  add: "bg-success/10 dark:bg-success/16",
  remove: "bg-destructive/10 dark:bg-destructive/16",
};
const SIGN: Record<CodeSign, string> = { context: "", add: "+", remove: "-" };
const numberClass = "px-2 text-right text-muted-foreground/70 tabular-nums select-none";

interface HistoryCodeLineProps {
  line: CodeLine;
  /** A diff numbers both sides and signs each line; a version read alone has one number and no signs. */
  numbered: boolean;
}

export const HistoryCodeLine = ({ line, numbered }: HistoryCodeLineProps) => (
  <div className={cn("col-span-full grid grid-cols-subgrid", ROW[line.sign])}>
    {numbered && <span className={numberClass}>{line.oldNo}</span>}
    <span className={numberClass}>{line.newNo}</span>
    {numbered && (
      <span className={cn("text-center select-none", line.sign === "add" && "text-success", line.sign === "remove" && "text-destructive")}>{SIGN[line.sign]}</span>
    )}
    <code className="min-h-[1lh] pr-3 break-words whitespace-pre-wrap">
      {line.parts.map((part, i) => (
        <span key={i}>
          {part.type === "context" && part.text}
          {part.type === "add" && <ins className="rounded-[2px] bg-success/30 no-underline">{part.text}</ins>}
          {part.type === "remove" && <del className="rounded-[2px] bg-destructive/30 no-underline">{part.text}</del>}
        </span>
      ))}
    </code>
  </div>
);
