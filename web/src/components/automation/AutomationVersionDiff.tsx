import { useMemo } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
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
    <SettingsCard
      id="pending-version"
      title="Pending vs active"
      description="A pushed version waits here until you merge it. The active one keeps running."
      footer={
        pending &&
        canUpdate && (
          <Button type="button" size="sm" onClick={() => merge.mutate(pending.id)} loading={merge.isPending}>
            Merge
          </Button>
        )
      }
    >
      {!pending && <EmptyRow>No pending version</EmptyRow>}
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
    </SettingsCard>
  );
};
