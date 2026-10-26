import { Check } from "lucide-react";

import { currentRowFillClass, CurrentRowEdge } from "@/components/CurrentRow";
import { RepoProviderMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { preferredCandidate, scanSummary, type Repo, type ScanResult } from "@/models/Repository";

interface WizardOpenRepositoryRowProps {
  repo: Repo;
  scan: ScanResult;
  // Absent when the scan found nothing to deploy; the manual Dockerfile path below takes over.
  onDeploy?: (() => void) | undefined;
}

// The picked repository, scanned: what the scan found, then the way on.
export const WizardOpenRepositoryRow = ({ repo, scan, onDeploy }: WizardOpenRepositoryRowProps) => {
  const candidate = preferredCandidate(scan);
  return (
    <div className={cn("relative px-2 py-3", currentRowFillClass)} aria-current="true">
      <CurrentRowEdge />
      <div className="flex items-center gap-3">
        <RepoProviderMark provider={repo.provider} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-medium">{repo.full_name}</span>
          <span className="flex min-w-0 items-center gap-1.5 font-mono text-xs text-muted-foreground">
            <span className="shrink-0">{scan.default_branch}</span>
            <span aria-hidden>·</span>
            {candidate && <Check className="size-3 shrink-0 text-success" strokeWidth={3} aria-hidden />}
            <span className="truncate">{scanSummary(candidate)}</span>
          </span>
        </span>
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-end gap-2">
        {onDeploy && (
          <Button type="button" size="sm" onClick={onDeploy}>
            Deploy it
          </Button>
        )}
      </div>
    </div>
  );
};
