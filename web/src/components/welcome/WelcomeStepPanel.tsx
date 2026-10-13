import { useState } from "react";

import { displayTitleClass } from "@/components/PageHeader";
import { WelcomeFinish } from "@/components/welcome/WelcomeFinish";
import { WelcomeStepBody } from "@/components/welcome/WelcomeStepBody";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { STEP_COPY, viewIndex } from "@/models/WelcomeWizard";
import { cn } from "@/lib/utils";

// Slides the step in 8px from the side it was reached from, the project wizard's 180ms; the finish view rises instead.
export const WelcomeStepPanel = () => {
  const { steps, view } = useWelcomeFlow();
  const index = viewIndex(steps, view);
  const [seen, setSeen] = useState(index);
  const [forward, setForward] = useState(true);
  if (index !== seen) {
    setForward(index > seen);
    setSeen(index);
  }

  return (
    <div
      key={view}
      className={cn(
        "animate-in fade-in-0 ease-out motion-reduce:animate-none",
        view === "finish" && "slide-in-from-bottom-1 duration-200",
        view !== "finish" && "duration-180",
        view !== "finish" && (forward ? "slide-in-from-right-2" : "slide-in-from-left-2"),
      )}
    >
      {view === "finish" && <WelcomeFinish />}
      {view !== "finish" && <h1 className={cn(displayTitleClass, "text-[2rem]")}>{STEP_COPY[view].title}</h1>}
      {view !== "finish" && <p className="mt-2 text-muted-foreground">{STEP_COPY[view].lead}</p>}
      {view !== "finish" && (
        <div className="mt-8">
          <WelcomeStepBody step={view} />
        </div>
      )}
    </div>
  );
};
