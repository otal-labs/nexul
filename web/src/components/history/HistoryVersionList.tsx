import type { KeyboardEvent } from "react";

import { HistoryFieldRow } from "@/components/history/HistoryFieldRow";
import { HistoryScrollFade } from "@/components/history/HistoryScrollFade";
import { HistoryVersionRow } from "@/components/history/HistoryVersionRow";
import { cn } from "@/lib/utils";
import type { HistoryFieldChange, HistoryVersionSummary } from "@/models/History";
import { keepsNamed, keptCounts, KEPT_UNNAMED_VERSIONS, railItems, restoredWhen } from "@/utils/HistoryVersionUtility";

const ARROWS: Record<string, number> = { ArrowUp: -1, ArrowDown: 1 };

interface HistoryVersionListProps {
  versions: HistoryVersionSummary[];
  changes?: HistoryFieldChange[] | undefined;
  selected: number;
  /** False where an item can never have a named version (memories, tickets, plays). */
  namesVersions?: boolean;
  onSelect: (version: number) => void;
}

// The rail: the versions newest first on a timeline, ending with the rule that decides what is kept. It is opaque and scrolls
// on its own; its gutter is reserved so the 200px of content at 768 never moves when the scrollbar comes in.
export const HistoryVersionList = ({ versions, changes, selected, namesVersions = true, onSelect }: HistoryVersionListProps) => {
  const counts = keptCounts(versions, namesVersions);
  const items = railItems(versions, changes);
  const picks = versions.map((v) => v.version);

  // Arrows move the pick through the versions and keep focus in the rail.
  const onKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    const direction = ARROWS[event.key];
    const row = event.target instanceof HTMLElement ? event.target.closest("[data-version]") : null;
    if (!direction || !row) return;
    const next = picks[picks.indexOf(Number(row.getAttribute("data-version"))) + direction];
    if (next === undefined) return;
    event.preventDefault();
    onSelect(next);
    event.currentTarget.querySelector<HTMLElement>(`[data-version="${next}"]`)?.focus();
  };

  return (
    <aside
      aria-label="Versions"
      onKeyDown={onKeyDown}
      className="flex min-h-0 w-62 shrink-0 flex-col overflow-y-auto border-l border-border bg-popover max-lg:w-58"
    >
      <div className="sticky top-0 z-10 flex h-11 shrink-0 items-center justify-between gap-1.5 border-b border-border bg-popover px-4">
        <h3 className="text-[11px] leading-4 font-semibold tracking-[0.1em] text-muted-foreground uppercase">Versions</h3>
        <span title={counts.long} className="min-w-0 truncate text-[11px] leading-4 tabular-nums text-muted-foreground">
          {counts.short}
          <span className="sr-only">. {counts.long}</span>
        </span>
      </div>
      <HistoryScrollFade edge="top" />
      <ul>
        {items.map((item, i) => (
          <li
            key={item.kind === "version" ? `v${item.summary.version}` : `c${item.change.id}`}
            className={cn("border-border", i > 0 && !(item.kind === "field" && items[i - 1]?.kind === "field") && "border-t")}
          >
            {item.kind === "version" && (
              <HistoryVersionRow
                summary={item.summary}
                selected={item.summary.version === selected}
                first={i === 0}
                last={i === items.length - 1}
                restoredWhen={restoredWhen(item.summary, versions)}
                onSelect={onSelect}
              />
            )}
            {item.kind === "field" && <HistoryFieldRow change={item.change} first={i === 0} last={i === items.length - 1} />}
          </li>
        ))}
        <li className="border-t border-border px-4 py-2 text-xs text-muted-foreground">Only the last {KEPT_UNNAMED_VERSIONS} versions are kept{keepsNamed(versions, namesVersions) ? ", plus named ones" : ""}.</li>
      </ul>
      <HistoryScrollFade edge="bottom" />
    </aside>
  );
};
