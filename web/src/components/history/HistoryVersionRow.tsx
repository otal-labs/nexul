import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { HistoryEditors } from "@/components/history/HistoryEditors";
import { cn } from "@/lib/utils";
import type { HistoryVersionSummary } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

interface HistoryVersionRowProps {
  summary: HistoryVersionSummary;
  selected: boolean;
  onSelect: (version: number) => void;
}

// One saved version in the list: its time, the name it was given, who made it and what it came through.
export const HistoryVersionRow = ({ summary, selected, onSelect }: HistoryVersionRowProps) => (
  <li>
    <button
      type="button"
      aria-current={selected}
      onClick={() => onSelect(summary.version)}
      className={cn(
        "inset-focus relative block w-full px-4 py-2.5 text-left transition-colors duration-150 ease-standard",
        !selected && "hover:bg-accent",
        selected && currentRowFillClass,
      )}
    >
      {selected && <CurrentRowEdge />}
      <span className="flex items-center gap-2 font-mono text-xs">
        {formatVersionTime(summary.created_at)}
        {summary.current && (
          <span className="rounded-full border border-border px-1.5 py-0.5 font-sans text-[10px] leading-none font-semibold tracking-wide text-muted-foreground uppercase">
            Current
          </span>
        )}
      </span>
      {summary.name && <span className="mt-0.5 block text-sm font-semibold">{summary.name}</span>}
      <HistoryEditors editorIds={summary.editor_ids} via={summary.via} />
    </button>
  </li>
);
