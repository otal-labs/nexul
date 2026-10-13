import { createContext, use, useState } from "react";
import { useNavigate } from "react-router";

import type { WelcomeProgress } from "@nexul/client-core/welcome";

import { useFinishWelcomeFirstRun, useSkipWelcomeStep } from "@/hooks/WelcomeHooks";
import { neighbourView, stepMarks, wizardSteps, type StepMark, type WizardStep, type WizardView } from "@/models/WelcomeWizard";

export interface WelcomeFlow {
  progress: WelcomeProgress;
  steps: WizardStep[];
  view: WizardView;
  marks: Record<WizardStep, StepMark>;
  open: (view: WizardView) => void;
  next: () => void;
  back: () => void;
  // Records the step on screen as skipped, then moves on.
  skip: () => void;
  skipping: boolean;
  // Leaving the wizard the first time is its first run, so it never opens by itself again.
  leave: (to: string) => void;
  leaving: boolean;
}

// The wizard's steps, footers and rail read the flow here instead of threading it through every step body.
export const WelcomeFlowContext = createContext<WelcomeFlow | null>(null);

export const useWelcomeFlow = (): WelcomeFlow => {
  const flow = use(WelcomeFlowContext);
  if (!flow) throw new Error("useWelcomeFlow is used outside the Welcome wizard");
  return flow;
};

// Whether a step was done when the wizard reached it, which shows what it found; one done while open settles live instead.
export const useDoneOnArrival = (step: WizardStep): boolean => {
  const { marks } = useWelcomeFlow();
  return useState(() => marks[step] === "done")[0];
};

// The URL's ?step is the whole navigation state, so Back and Forward in the browser walk the steps too.
export const useWelcomeFlowState = (progress: WelcomeProgress, view: WizardView): WelcomeFlow => {
  const navigate = useNavigate();
  const skipStep = useSkipWelcomeStep();
  const finish = useFinishWelcomeFirstRun();
  const steps = wizardSteps(progress);
  const open = (to: WizardView) => navigate({ search: `?step=${to}` });
  const next = () => open(neighbourView(steps, view, 1));

  return {
    progress,
    steps,
    view,
    marks: stepMarks(progress, view),
    open,
    next,
    back: () => open(neighbourView(steps, view, -1)),
    skip: () => {
      if (view === "profile" || view === "finish") return next();
      skipStep.mutate(view, { onSuccess: next });
    },
    skipping: skipStep.isPending,
    leave: (to) => {
      if (progress.first_run_done) return navigate(to);
      finish.mutate(undefined, { onSuccess: () => navigate(to, { replace: true }) });
    },
    leaving: finish.isPending,
  };
};
