import { EnterList } from "@/components/EnterList";
import { DeployStepRow } from "@/components/deploy/DeployStepRow";
import type { DeployStep } from "@/utils/DeployLogUtility";

interface DeployStepListProps {
  title: string;
  steps: (Pick<DeployStep, "label" | "state" | "durationMs"> & { key: string })[];
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
      <EnterList as="ol" aria-label="Steps">
        {steps.map((step) => (
          <DeployStepRow key={step.key} step={step} />
        ))}
      </EnterList>
    </div>
  );
};
