import { RunDot, RegenerateButton, useRunSteps, type RunStep } from "@/components/memory/prototype/InterviewPrototypeRun";
import { cn } from "@/lib/utils";

const TimelineStep = ({ step, last }: { step: RunStep; last: boolean }) => (
  <li className="flex gap-3">
    <span className="flex flex-col items-center">
      <RunDot tone={step.tone} className="mt-1.5" />
      {!last && <span className={cn("mt-1 min-h-4 w-px flex-1", step.tone === "done" ? "bg-foreground/40" : "bg-border")} />}
    </span>
    <span className={cn("min-w-0", !last && "pb-3")}>
      <span className={cn("block text-sm", step.tone === "upcoming" ? "text-muted-foreground" : "text-foreground")}>{step.label}</span>
      {step.detail && <span className="block text-xs text-muted-foreground">{step.detail}</span>}
    </span>
  </li>
);

// Variant A's side card: the run as a short timeline, Regenerate once a memory exists, and the help line.
export const InterviewPrototypeTimeline = ({ className }: { className?: string }) => {
  const { steps } = useRunSteps();
  return (
    <aside className={cn("h-fit rounded-lg border border-border bg-card p-4", className)}>
      <p className="text-sm font-medium">Interview run</p>
      <ol className="mt-3">
        {steps.map((step, i) => (
          <TimelineStep key={step.label} step={step} last={i === steps.length - 1} />
        ))}
      </ol>
      <RegenerateButton className="mt-4 w-full" />
      <p className="mt-4 border-t border-border pt-3 text-xs text-muted-foreground">
        The agent asks before it records anything it read in the code.
      </p>
    </aside>
  );
};
