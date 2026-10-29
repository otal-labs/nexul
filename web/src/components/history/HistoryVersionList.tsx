import { HistoryVersionRow } from "@/components/history/HistoryVersionRow";
import { Microheader } from "@/components/Microheader";
import type { HistoryVersionSummary } from "@/models/History";

interface HistoryVersionListProps {
  versions: HistoryVersionSummary[];
  selected: number;
  onSelect: (version: number) => void;
}

// Newest first, ending with the rule that decides what is kept.
export const HistoryVersionList = ({ versions, selected, onSelect }: HistoryVersionListProps) => (
  <aside className="flex min-h-0 flex-col overflow-y-auto border-l border-border">
    <Microheader className="px-4 pt-3.5 pb-2">Versions</Microheader>
    <ul className="divide-y divide-border border-t border-border">
      {versions.map((summary) => (
        <HistoryVersionRow key={summary.version} summary={summary} selected={summary.version === selected} onSelect={onSelect} />
      ))}
    </ul>
    <p className="mt-auto border-t border-border px-4 py-2.5 text-xs text-muted-foreground">Only the last 10 versions are kept, plus named ones.</p>
  </aside>
);
