import { ImportFailures } from "@/components/import/ImportFailures";
import { ImportStateStatus } from "@/components/import/ImportStateStatus";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { importSourceLabel, ImportState, type Import } from "@/models/Import";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface ImportHistoryRowProps {
  imp: Import;
}

// What one ended import read, where it went, how it ended and what it brought in.
export const ImportHistoryRow = ({ imp }: ImportHistoryRowProps) => {
  const { data: projects } = useFetchProjects();
  const project = projects?.find((p) => p.id === imp.project_id)?.name;
  const counts = `${imp.done.toLocaleString()} of ${imp.total.toLocaleString()} in${imp.report.skipped > 0 ? `, ${imp.report.skipped.toLocaleString()} already here` : ""}${imp.report.unsupported > 0 ? `, ${imp.report.unsupported.toLocaleString()} blocks as text or links` : ""}`;
  return (
    <li className="py-3">
      <div className="flex min-w-0 items-baseline justify-between gap-3">
        <p className="min-w-0 truncate text-sm">
          {importSourceLabel(imp.source)}
          {imp.from && <span className="font-mono"> {imp.from}</span>}
          {project && <span className="text-muted-foreground"> into {project}</span>}
        </p>
        <span className="shrink-0 text-xs text-muted-foreground">{formatRelativeTime(imp.updated_at)}</span>
      </div>
      <ImportStateStatus state={imp.state} detail={imp.report.error || counts} />
      {imp.state !== ImportState.Cancelled && imp.report.failures.length > 0 && <ImportFailures id={imp.id} />}
    </li>
  );
};
