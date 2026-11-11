import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryCodeDiff } from "@/components/history/HistoryCodeDiff";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { automationVersionUrl } from "@/hooks/AutomationVersionHooks";
import { useHistoryCompare } from "@/hooks/HistoryCompareHooks";

interface AutomationVersionCodeProps {
  automationId: string;
  sequence: number;
  /** Diffs against this version; without it the whole file shows. */
  against?: number | undefined;
}

// The History dialog's code renderer, inline: numbered lines, and folded context around each change in a diff.
export const AutomationVersionCode = ({ automationId, sequence, against }: AutomationVersionCodeProps) => {
  const compare = useHistoryCompare((version) => automationVersionUrl(automationId, version), sequence, against, against !== undefined, "code");

  return (
    <div className="scroll-edge max-h-96 overflow-y-auto rounded-md">
      {compare.isPending && <LoadingDisplay />}
      {compare.error && <ErrorDisplay error={compare.error} />}
      {compare.runs && <HistoryCodeDiff runs={compare.runs} changes={compare.changes} />}
    </div>
  );
};
