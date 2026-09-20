interface TrailStepDetailProps {
  detail: string;
}

interface DetailBlocks {
  input?: unknown;
  result?: unknown;
}

const pretty = (value: unknown): string => {
  if (typeof value === "string") return value;
  return JSON.stringify(value, null, 2);
};

// A tool result's content is what the reader wants; the wrapping envelope is noise.
const resultText = (result: unknown): string => {
  if (typeof result === "object" && result !== null && "content" in result && typeof result.content === "string") return result.content;
  return pretty(result);
};

// Detail is {input, result} JSON for a tool step and plain text otherwise; either way it is shown as-is, never re-shaped.
const parseBlocks = (detail: string): DetailBlocks | null => {
  try {
    const parsed: unknown = JSON.parse(detail);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return null;
    if (!("input" in parsed) && !("result" in parsed)) return null;
    return parsed as DetailBlocks;
  } catch {
    return null;
  }
};

const blockClass = "max-h-72 overflow-auto rounded-md border border-border bg-background p-2 font-mono text-[10px] leading-relaxed whitespace-pre-wrap break-all";
const labelClass = "font-mono text-[10px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// The expanded half of a step row: the arguments and, when present, the result, each in its own mono block.
export const TrailStepDetail = ({ detail }: TrailStepDetailProps) => {
  const blocks = parseBlocks(detail);
  return (
    <div className="space-y-2 py-1 pr-2 pl-11.5">
      {blocks === null && <pre className={blockClass}>{detail}</pre>}
      {blocks !== null && blocks.input !== undefined && (
        <div className="space-y-1">
          <p className={labelClass}>Arguments</p>
          <pre className={blockClass}>{pretty(blocks.input)}</pre>
        </div>
      )}
      {blocks !== null && blocks.result !== undefined && (
        <div className="space-y-1">
          <p className={labelClass}>Result</p>
          <pre className={blockClass}>{resultText(blocks.result)}</pre>
        </div>
      )}
    </div>
  );
};
