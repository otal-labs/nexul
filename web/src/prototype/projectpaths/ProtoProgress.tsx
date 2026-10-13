import { WizardProgressItem, type WizardProgressState } from "@/components/wizard/WizardProgressItem";
import { wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

import { STEPS } from "./mock";

interface ProtoProgressProps {
  states: Record<WizardStepId, WizardProgressState>;
  // Steps whose incoming connector stays unfilled, so a jump reads as a gap in the line.
  gaps?: WizardStepId[];
  // Steps drawn faded: not part of this project's path for now.
  faded?: WizardStepId[];
}

export const ProtoProgress = ({ states, gaps = [], faded = [] }: ProtoProgressProps) => {
  const current = STEPS.findIndex((id) => states[id] === "current");
  const fadeCss = faded
    .map((id) => `.pp-progress > li:nth-child(${STEPS.indexOf(id) + 1}) > :not(span:first-child) { opacity: 0.45; }`)
    .join("\n");
  return (
    <div className="@container">
      {fadeCss && <style>{fadeCss}</style>}
      <ol aria-label="Project wizard steps" className="pp-progress flex">
        {STEPS.map((id, index) => (
          <WizardProgressItem
            key={id}
            label={wizardStepLabel(id, false)}
            state={states[id]}
            first={index === 0}
            reached={index <= current && !gaps.includes(id)}
            onSelect={() => undefined}
          />
        ))}
      </ol>
      <p className="mt-3 text-center font-mono text-xs @2xl:hidden">
        <span className="font-bold text-foreground">{wizardStepLabel(STEPS[current]!, false)}</span>{" "}
        <span className="text-muted-foreground">
          {current + 1} / {STEPS.length}
        </span>
      </p>
    </div>
  );
};
