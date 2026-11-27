import { personLabel } from "@nexul/client-core/person";

import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
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
  const headline = summary.name || `Version ${summary.version}`;
  const note = summary.name || !summary.builtin_default ? "" : "Built-in default";
  const date = formatVersionTime(summary.created_at);
  const [editor] = people;
  // Who saved it, in a phrase: an editor's name, what an automatic save came through, or that it was saved on its own.
  const made = (() => {
    if (editor) return `edited by ${personLabel(editor)}${people.length > 1 ? ` +${people.length - 1}` : ""}`;
    return summary.via ? viaLabel(summary.via) : "saved automatically";
  })();
  const pending = summary.status === "pending";
  const label = [
    date,
    people.map(personLabel).join(", "),
    summary.via && viaLabel(summary.via),
    headline,
    note,
    size?.exact,
    summary.current && "current version",
    pending && "pending version",
    restoredWhen && `Restored from ${restoredWhen}`,
  ]
    .filter(Boolean)
    .join(", ");

  // The button lies under the row's content rather than around it, so the author's count can be a button of its own.
  return (
    <div className={cn("relative flex w-full gap-1.5 px-4 py-2 text-left [&>:not(button)]:pointer-events-none", selected && currentRowFillClass)}>
      <button
        type="button"
        data-version={summary.version}
        aria-current={selected}
        aria-label={label}
        onClick={() => onSelect(summary.version)}
        className={cn("inset-focus absolute inset-0 transition-colors duration-150 ease-standard", !selected && "hover:bg-accent")}
      />
      {selected && <CurrentRowEdge />}
      <HistorySpine node={summary.name || summary.builtin_default ? "named" : "save"} selected={selected} first={first} last={last} />
      <span className="relative min-w-0 flex-1">
        <span className="flex items-baseline justify-between gap-2">
          <span className="min-w-0 line-clamp-2 text-row leading-[1.3] font-semibold text-balance [overflow-wrap:anywhere]" title={headline}>
            {headline}
          </span>
          {size && (
            <span className="shrink-0 font-mono text-xs tabular-nums" title={size.exact}>
              {size.added > 0 && <span className="text-success">+{cappedCount(size.added)}</span>}
              {size.added > 0 && size.removed > 0 && " "}
              {size.removed > 0 && <span className="text-destructive">−{cappedCount(size.removed)}</span>}
            </span>
          )}
        </span>
        {note && <span className="mb-0.5 block text-xs text-muted-foreground">{note}</span>}
        <span className="block font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums" title={made}>
          {date}
        </span>
        {(summary.current || pending || restoredWhen) && (
          <span className="mt-1 block text-micro leading-4 text-muted-foreground">
            {summary.current && <span className="block">Current</span>}
            {pending && <span className="block">Pending</span>}
            {restoredWhen && (
              <span className="block truncate">
                <span className="text-foreground">Restored from </span>
                {restoredWhen}
              </span>
            )}
          </span>
        )}
      </span>
    </div>
  );
};
