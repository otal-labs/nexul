import { lightsDot, type WelcomeProgress, type WelcomeStepProgress } from "@nexul/client-core/welcome";

export interface WelcomeMenuSummary {
  // Steps not done, in the wizard's order.
  open: WelcomeStepProgress[];
  // Steps left to do; skipped ones are not counted.
  left: number;
  done: number;
  total: number;
}

// Projects before anyone gives it is something a person can't do alone, so the menu leaves it out.
export const welcomeMenuSummary = (progress: WelcomeProgress): WelcomeMenuSummary => {
  const steps = progress.steps.filter((step) => step.state !== "left" || lightsDot(step));
  return {
    open: steps.filter((step) => step.state !== "done"),
    left: steps.filter((step) => step.state === "left").length,
    done: steps.filter((step) => step.state === "done").length,
    total: steps.length,
  };
};
