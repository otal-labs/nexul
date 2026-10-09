import { EASE_OUT, SPRING_POP, prefersReducedMotion } from "@/lib/motion";
import type { DeployStepState } from "@/utils/DeployLogUtility";

// Rail-led like the DNS stepper: mark pops, rail draws to the next rung, then the next node lights.
export const playStep = (row: HTMLElement | null, from: DeployStepState, to: DeployStepState) => {
  if (!row || from === to || typeof row.animate !== "function") return;
  const node = row.querySelector<HTMLElement>("[data-step-node]");
  const rail = row.querySelector<HTMLElement>("[data-step-rail]");
  if (prefersReducedMotion()) {
    node?.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 150, easing: EASE_OUT });
    return;
  }
  if (to === "done" || to === "failed") {
    node?.animate([{ transform: "scale(0.6)", opacity: 0.4 }, { transform: "none", opacity: 1 }], { duration: 350, easing: SPRING_POP });
    rail?.animate([{ transform: "scaleY(0)" }, { transform: "none" }], { duration: 260, easing: EASE_OUT });
    return;
  }
  if (to === "active")
    node?.animate([{ transform: "scale(0.85)", opacity: 0 }, { transform: "none", opacity: 1 }], { duration: 200, delay: 180, easing: EASE_OUT, fill: "backwards" });
};
