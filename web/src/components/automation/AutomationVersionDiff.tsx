import { useMemo } from "react";

import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Button } from "@/components/ui/button";
import { useMergeAutomationVersion } from "@/hooks/AutomationVersionHooks";
import type { AutomationVersionDiff as VersionDiffData } from "@/models/AutomationVersion";
import { cn } from "@/lib/utils";
import { diffLines } from "@/utils/DiffUtility";

interface AutomationVersionDiffProps {
  automationId: string;
  diff: VersionDiffData;
  canUpdate: boolean;
}

const lineClass = (type: "context" | "add" | "remove") =>
  cn(
    "block px-3",
    type === "add" && "bg-success/10 text-success before:content-['+_']",
    type === "remove" && "bg-destructive/10 text-destructive before:content-['-_']",
    type === "context" && "text-muted-foreground before:content-['___']",
  );

// ponytail: lightweight LCS line-diff, not a library — code files are short enough the cost never matters yet.
export const AutomationVersionDiff = ({ automationId, diff, canUpdate }: AutomationVersionDiffProps) => {
  const merge = useMergeAutomationVersion(automationId);
  const pending = diff.pending;
  const lines = useMemo(() => diffLines(diff.active?.code ?? "", pending?.code ?? ""), [diff.active, pending]);

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">Pending vs active</h2>
        {pending && canUpdate && (
          <Button type="button" size="sm" onClick={() => merge.mutate(pending.id)} disabled={merge.isPending}>
            {merge.isPending ? "Merging…" : "Merge"}
          </Button>
        )}
      </div>
      {!pending && <NoDataDisplay message="No pending version" size="compact" />}
      {pending && (
        <div className="terminal-window">
          <div className="terminal-window__bar">
            <span className="terminal-window__dot" aria-hidden />
            <span className="terminal-window__dot" aria-hidden />
            <span className="terminal-window__dot" aria-hidden />
            <span className="terminal-window__title">
              #{diff.active?.sequence ?? 0} → #{pending.sequence}
            </span>
          </div>
          <pre className="terminal-window__body max-h-96 overflow-y-auto whitespace-pre-wrap break-words">
            {lines.map((line, index) => (
              <span key={index} className={lineClass(line.type)}>
                {line.text || " "}
              </span>
            ))}
          </pre>
        </div>
      )}
    </section>
  );
};
