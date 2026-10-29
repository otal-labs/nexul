import { HistoryBlockRow } from "@/components/history/HistoryBlockRow";
import { HistoryText } from "@/components/history/HistoryText";
import { displayTitleClass } from "@/components/PageHeader";
import type { DiffSegment } from "@/utils/DiffUtility";
import type { HistoryRow } from "@/utils/HistoryBlocksUtility";

interface HistoryDocumentProps {
  title: DiffSegment[];
  rows: HistoryRow[];
}

export const HistoryDocument = ({ title, rows }: HistoryDocumentProps) => (
  <>
    <h3 className={`${displayTitleClass} mb-4 text-2xl`}>
      <HistoryText parts={title} />
    </h3>
    <div className="prose-rich text-sm">
      {rows.map((row, i) => (
        <HistoryBlockRow key={i} row={row} />
      ))}
    </div>
  </>
);
