import { cn } from "@/lib/utils";
import type { DiffSegment } from "@/utils/DiffUtility";

// A struck line in the destructive hue, so removal never rests on colour alone.
export const strikeClass = "line-through decoration-destructive decoration-[1.5px]";

interface HistoryPartProps {
  part: DiffSegment;
  /** A removed run that an added one follows keeps a hairline gap, so the two tints never read as one mark. */
  beforeAdded: boolean;
}

// Word marks carry no horizontal padding, so a mark never moves a line break; the spaces at a run's ends stay outside the
// mark, so a tint never covers one.
const HistoryPart = ({ part, beforeAdded }: HistoryPartProps) => {
  const [, lead = "", core = "", trail = ""] = /^(\s*)([\s\S]*?)(\s*)$/.exec(part.text) ?? [];
  return (
    <>
      {part.type === "context" && part.text}
      {part.type !== "context" && lead}
      {part.type === "add" && core && <ins className="rounded-[2px] bg-success/22 text-foreground no-underline">{core}</ins>}
      {part.type === "remove" && core && <del className={cn("rounded-[2px] bg-destructive/22 text-muted-foreground", strikeClass, beforeAdded && "mr-px")}>{core}</del>}
      {part.type !== "context" && trail}
    </>
  );
};

interface HistoryTextProps {
  parts: DiffSegment[];
}

export const HistoryText = ({ parts }: HistoryTextProps) =>
  parts.map((part, i) => <HistoryPart key={i} part={part} beforeAdded={part.type === "remove" && parts[i + 1]?.type === "add"} />);
