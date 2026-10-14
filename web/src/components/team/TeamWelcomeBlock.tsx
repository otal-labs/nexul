import { Check } from "lucide-react";

import { Microheader } from "@/components/Microheader";
import { TeamWelcomeStepRow } from "@/components/team/TeamWelcomeStepRow";
import type { TeamPerson } from "@/models/Team";
import { TEAM_STEP_NAMES, teamProgress } from "@/utils/TeamProgressUtility";

// The person's Welcome wizard: what is left or skipped as rows, what is done folded into one line under them.
export const TeamWelcomeBlock = ({ person }: { person: TeamPerson }) => {
  const progress = teamProgress(person);
  if (!progress) return null;
  const open = progress.steps.filter((step) => step.state !== "done");
  const finished = progress.steps.filter((step) => step.state === "done").map((step) => TEAM_STEP_NAMES[step.key]).join(", ");
  return (
    <section aria-labelledby="team-welcome-title" className="space-y-2 px-6 pb-4">
      <div className="flex items-baseline justify-between gap-3">
        <Microheader id="team-welcome-title">Welcome wizard</Microheader>
        <span className="font-mono text-xs text-muted-foreground tabular-nums">
          {progress.done} of {progress.steps.length} set up
        </span>
      </div>
      {open.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {open.map((step) => (
            <TeamWelcomeStepRow key={step.key} step={step} />
          ))}
        </ul>
      )}
      {progress.done > 0 && (
        <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Check aria-hidden className="size-3.5 text-success" />
          {open.length === 0 && <span>Every step done: {finished}</span>}
          {open.length > 0 && <span>Done: {finished}</span>}
        </p>
      )}
    </section>
  );
};
