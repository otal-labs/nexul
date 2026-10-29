import type { DiffSegment } from "@/utils/DiffUtility";

interface HistoryPartProps {
  part: DiffSegment;
}

// Added words are tinted in the success hue, removed words struck through in the destructive one.
const HistoryPart = ({ part }: HistoryPartProps) => (
  <>
    {part.type === "context" && part.text}
    {part.type === "add" && <ins className="rounded-sm bg-success/15 px-0.5 text-success no-underline">{part.text}</ins>}
    {part.type === "remove" && <del className="rounded-sm bg-destructive/10 px-0.5 text-destructive decoration-destructive/70">{part.text}</del>}
  </>
);

interface HistoryTextProps {
  parts: DiffSegment[];
}

export const HistoryText = ({ parts }: HistoryTextProps) => parts.map((part, i) => <HistoryPart key={i} part={part} />);
