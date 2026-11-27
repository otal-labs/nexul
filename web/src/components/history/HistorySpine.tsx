import { cn } from "@/lib/utils";

interface HistorySpineProps {
  /** A node is one hollow ring, filled while its version is picked; a dot is a field change. */
  node: "save" | "named" | "field";
  selected?: boolean;
  first: boolean;
  last: boolean;
}

// The rail's timeline: a hairline spine runs down every row, joined across the row dividers, with a node on it; the picked row's node is the one filled, in the accent.
export const HistorySpine = ({ node, selected = false, first, last }: HistorySpineProps) => (
  <span aria-hidden className="relative -ml-0.5 w-3 shrink-0 self-stretch">
    <span
      className={cn(
        "absolute left-1/2 w-px -translate-x-1/2 bg-border",
        first ? "top-2" : "-top-[9px]",
        last ? "bottom-[calc(100%-0.5rem)]" : "-bottom-[9px]",
      )}
    />
    {node === "field" && <span className="absolute top-[11px] left-1/2 size-1.5 -translate-x-1/2 rounded-full bg-muted-foreground" />}
    {node !== "field" && (
      <span
        className={cn(
          "absolute top-[3px] left-1/2 size-2.5 -translate-x-1/2 rounded-full border-2",
          selected ? "border-brand bg-brand" : "border-muted-foreground/70 bg-popover",
        )}
      />
    )}
  </span>
);
