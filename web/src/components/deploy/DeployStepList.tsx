import { DeployStepRow } from "@/components/deploy/DeployStepRow";
import type { DeployStep } from "@/utils/DeployLogUtility";

interface DeployStepListProps {
  title: string;
  steps: DeployStep[];
}

// The log panel itself is aria-live="off"; this one polite line is what a screen reader hears on a step change.
export const DeployStepList = ({ title, steps }: DeployStepListProps) => {
  const active = steps.find((s) => s.state === "active");
  return (
    <div>
      <p role="status" className="sr-only">
        {title}
        {active && `: ${active.label}`}
      </p>
      <ol className="divide-y divide-border border-y border-border">
        {steps.map((step) => (
          <DeployStepRow key={step.key} step={step} />
        ))}
      </ol>
    </div>
  );
};
