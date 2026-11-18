import { useImport } from "@/hooks/ImportHooks";

interface ImportProgressProps {
  id: string;
}

// Records handled out of the scanned total as a figure and a meter; failures say so beside it.
export const ImportProgress = ({ id }: ImportProgressProps) => {
  const imp = useImport(id);
  if (!imp) return null;
  const handled = imp.done + imp.failed;
  const share = imp.total > 0 ? Math.min(100, Math.round((handled / imp.total) * 100)) : 0;
  return (
    <div className="space-y-2">
      <p className="flex items-baseline gap-2 text-sm">
        <span className="text-2xl font-semibold tabular-nums">
          {handled.toLocaleString()}/{imp.total.toLocaleString()}
        </span>
        <span className="text-muted-foreground">
          handled
          {imp.failed > 0 && ` · ${imp.failed.toLocaleString()} failed`}
          {imp.report.skipped > 0 && ` · ${imp.report.skipped.toLocaleString()} already here`}
        </span>
      </p>
      <div
        role="progressbar"
        aria-label="Import progress"
        aria-valuemin={0}
        aria-valuemax={imp.total}
        aria-valuenow={handled}
        className="h-1 overflow-hidden rounded-full bg-muted"
      >
        <div className="h-full bg-brand transition-[width] duration-300 ease-standard" style={{ width: `${share}%` }} />
      </div>
    </div>
  );
};
