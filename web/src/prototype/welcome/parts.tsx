import { Check, Minus } from "lucide-react";

import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { STEPS, useWelcomeProto, type StepKey, type StepState } from "./model";
import { ComputerStep } from "./steps/ComputerStep";
import { DeployServerStep } from "./steps/DeployServerStep";
import { GitHubStep } from "./steps/GitHubStep";
import { PhoneStep } from "./steps/PhoneStep";
import { ProfileStep } from "./steps/ProfileStep";
import { ProjectsStep } from "./steps/ProjectsStep";

export const StepBody = ({ step }: { step: StepKey }) => (
  <>
    {step === "profile" && <ProfileStep />}
    {step === "github" && <GitHubStep />}
    {step === "computer" && <ComputerStep />}
    {step === "projects" && <ProjectsStep />}
    {step === "deploy_server" && <DeployServerStep />}
    {step === "phone" && <PhoneStep />}
  </>
);

export type NodeState = StepState | "current";

const node = "relative z-10 flex size-5 shrink-0 items-center justify-center rounded-full bg-background";

// The project wizard's progress node: a success check, a dashed ring with a dash, a hollow ring, or the brand ring.
export const StepNode = ({ state }: { state: NodeState }) => (
  <span
    aria-hidden
    className={cn(
      node,
      state === "current" && "border-2 border-brand",
      state === "done" && "border border-border text-success",
      state === "skipped" && "border border-dashed border-muted-foreground/60 text-muted-foreground",
      state === "todo" && "border border-muted-foreground/40",
    )}
  >
    {state === "current" && <span className="size-2 rounded-full bg-brand" />}
    {state === "done" && <Check key="d" className="pop-in size-3" strokeWidth={3} />}
    {state === "skipped" && <Minus className="size-3" strokeWidth={3} />}
  </span>
);

export const STATE_WORD: Record<StepState, string> = { done: "Done", skipped: "Skipped", todo: "" };

interface FinishViewProps {
  onOpen: (step: StepKey) => void;
  owner: boolean;
}

// How the wizard ends: what was left for later as rows with Open, and one way into the app.
export const FinishView = ({ onOpen, owner }: FinishViewProps) => {
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  const later = order.filter((k) => states[k] !== "done");
  return (
    <div className="animate-in fade-in-0 slide-in-from-bottom-1 duration-200 ease-out">
      {later.length === 0 && <WizardConfirmation title="You're set up" subtitle="Everything on your list is done." />}
      {later.length > 0 && (
        <>
          <h1 className={cn(displayTitleClass, "text-[2rem]")}>
            {later.length === 1 ? "One thing left for later" : `${later.length} things left for later`}
          </h1>
          <p className="mt-2 text-muted-foreground">They wait under Welcome in your account menu, with a red dot until they're done.</p>
          <ul className="mt-8 divide-y divide-border border-y border-border">
            {later.map((k) => (
              <li key={k} className="flex items-center gap-3 py-2.5">
                <StepNode state={states[k]} />
                <span className="flex-1 text-sm">{STEPS[k].name}</span>
                <span className="font-mono text-xs text-muted-foreground">{STATE_WORD[states[k]] || "To do"}</span>
                <Button variant="ghost" size="sm" onClick={() => onOpen(k)}>
                  Open
                </Button>
              </li>
            ))}
          </ul>
        </>
      )}
      <div className="flex justify-end pt-8">
        <Button>{owner ? "Create your first project" : "Go to Northwind"}</Button>
      </div>
    </div>
  );
};
