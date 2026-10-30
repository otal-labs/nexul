import { useMemo } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ActionButton } from "@/components/action/ActionButton";
import { useMergeAutomationVersion } from "@/hooks/AutomationVersionHooks";
import { diffSides, type AutomationVersionDiff as VersionDiffData } from "@/models/AutomationVersion";
import { cn } from "@/lib/utils";
import { diffLines } from "@/utils/DiffUtility";

interface AutomationVersionDiffProps {
  automationId: string;
  diff: VersionDiffData;
}

const lineClass = (type: "context" | "add" | "remove") =>
  cn(
    "block pr-3 pl-[calc(0.75rem+2ch)] -indent-[2ch]",
    type === "add" && "bg-success/10 text-success before:content-['+_']",
    type === "remove" && "bg-destructive/10 text-destructive before:content-['-_']",
    type === "context" && "text-muted-foreground before:content-['___']",
  );

// ponytail: lightweight LCS line-diff, not a library — code files are short enough the cost never matters yet.
export const AutomationVersionDiff = ({ automationId, diff }: AutomationVersionDiffProps) => {
  const merge = useMergeAutomationVersion(automationId);
  const pending = diff.pending;
  const sides = useMemo(() => diffSides(diff), [diff]);
  const lines = useMemo(() => diffLines(sides.before, sides.after), [sides]);

  return (
    <SettingsCard
      id="pending-version"
      title="Pending vs active"
      description={
        pending
          ? `Merging makes #${pending.sequence} active${diff.active ? `; #${diff.active.sequence} stays to roll back to` : ""}.`
          : "A pushed version waits here until you merge it. The active one keeps running."
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
              {sides.bySource ? "" : " · bundled code"}
            </span>
            <ActionButton
              permission="automations:write"
              verb="merge automation versions"
              type="button"
              size="sm"
              className="ml-auto"
              onClick={() => merge.mutate(pending.id)}
              loading={merge.isPending}
            >
              Merge
            </ActionButton>
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
