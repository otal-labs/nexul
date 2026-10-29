import { personLabel } from "@nexul/client-core/person";

import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { HistoryAuthor } from "@/components/history/HistoryAuthor";
import { HistorySpine } from "@/components/history/HistorySpine";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import type { HistoryVersionSummary } from "@/models/History";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { formatVersionTime } from "@/utils/TimeUtility";
import { cappedCount, versionSize, viaLabel } from "@/utils/HistoryVersionUtility";

interface HistoryVersionRowProps {
  summary: HistoryVersionSummary;
  selected: boolean;
  first: boolean;
  last: boolean;
  /** The time (or number) of the version a restore copied, else empty. */
  restoredWhen: string;
  onSelect: (version: number) => void;
}

// One saved version on the rail's timeline: its time and size, who made it and what it came through; a named version leads
// with its name, and "Current" or the restore it was closes the row.
export const HistoryVersionRow = ({ summary, selected, first, last, restoredWhen, onSelect }: HistoryVersionRowProps) => {
  const person = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const people = summary.editor_ids.map(person);
  const size = versionSize(summary);
  const date = formatVersionTime(summary.created_at);
  const label = [
    date,
    people.map(personLabel).join(", "),
    summary.via && viaLabel(summary.via),
    summary.name,
    size?.exact,
    summary.current && "current version",
    restoredWhen && `Restored from ${restoredWhen}`,
  ]
    .filter(Boolean)
    .join(", ");

  return (
    <button
      type="button"
      data-version={summary.version}
      aria-current={selected}
      aria-label={label}
      onClick={() => onSelect(summary.version)}
      className={cn(
        "inset-focus relative flex w-full gap-1.5 px-4 py-2 text-left transition-colors duration-150 ease-standard",
        !selected && "hover:bg-accent",
        selected && currentRowFillClass,
      )}
    >
      {selected && <CurrentRowEdge />}
      <HistorySpine node={summary.name ? "named" : "save"} selected={selected} first={first} last={last} />
      <span className="min-w-0 flex-1">
        {summary.name && (
          <span className="mb-0.5 line-clamp-2 text-[13px] leading-[1.3] font-semibold text-balance" title={summary.name}>
            {summary.name}
          </span>
        )}
        <span className="flex items-baseline justify-between gap-1 font-mono text-xs tabular-nums">
          <span className="min-w-0 truncate">{date}</span>
          {size && (
            <span className="shrink-0" title={size.exact}>
              {size.added > 0 && <span className="text-success">+{cappedCount(size.added)}</span>}
              {size.added > 0 && size.removed > 0 && " "}
              {size.removed > 0 && <span className="text-destructive">−{cappedCount(size.removed)}</span>}
            </span>
          )}
        </span>
        <HistoryAuthor people={people} via={summary.via} />
        {(summary.current || restoredWhen) && (
          <span className="mt-1 block text-[11px] leading-4 text-muted-foreground">
            {summary.current && <span className="block">Current</span>}
            {restoredWhen && (
              <span className="block truncate">
                <span className="text-foreground">Restored from </span>
                {restoredWhen}
              </span>
            )}
          </span>
        )}
      </span>
    </button>
  );
};
