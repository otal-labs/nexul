import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { csvPreviewColumns, csvPreviewRows, useAttachmentCsv } from "@/hooks/AttachmentHooks";
import type { Attachment } from "@/models/Attachment";
import { cn } from "@/lib/utils";

interface CsvRowProps {
  cells: string[];
  header?: boolean;
}

const CsvRow = ({ cells, header = false }: CsvRowProps) => {
  const Cell = header ? "th" : "td";
  return (
    <tr className="border-b border-border last:border-b-0">
      {cells.map((cell, i) => (
        <Cell key={i} className={cn("px-2 py-1 text-left align-top font-normal whitespace-pre-wrap", header && "font-medium text-foreground")}>
          {cell}
        </Cell>
      ))}
    </tr>
  );
};

export const AttachmentCsvPreview = ({ attachment }: { attachment: Attachment }) => {
  const { data: table, error, isPending } = useAttachmentCsv(attachment);
  const [header, ...rows] = table?.rows ?? [];

  return (
    <div className="min-h-0 space-y-2">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {table && !header && <EmptyRow flush>This file is empty.</EmptyRow>}
      {header && (
        <div className="max-h-[70vh] overflow-auto rounded-md border border-border bg-surface-2">
          <table className="w-max min-w-full font-mono text-xs text-muted-foreground">
            <thead className="sticky top-0 bg-surface-2">
              <CsvRow cells={header} header />
            </thead>
            <tbody>
              {rows.map((cells, i) => (
                <CsvRow key={i} cells={cells} />
              ))}
            </tbody>
          </table>
        </div>
      )}
      {table?.truncated && (
        <p className="text-xs text-muted-foreground">Showing up to {csvPreviewRows} rows and {csvPreviewColumns} columns. Download it for the rest.</p>
      )}
    </div>
  );
};
