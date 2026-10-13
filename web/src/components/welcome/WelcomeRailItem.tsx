import { WelcomeStepNode } from "@/components/welcome/WelcomeStepNode";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";
import { MARK_WORD, STEP_NAMES, type WizardStep } from "@/models/WelcomeWizard";
import { cn } from "@/lib/utils";

interface WelcomeRailItemProps {
  step: WizardStep;
  last: boolean;
}

// Every step is a button, whatever happened there, so any step opens at any time.
export const WelcomeRailItem = ({ step, last }: WelcomeRailItemProps) => {
  const { view, marks, open } = useWelcomeFlow();
  const current = step === view;
  const mark = marks[step];
  return (
    <li className="relative">
      {!last && <span aria-hidden className="absolute top-7 bottom-0 left-[9.5px] w-px bg-border" />}
      <button
        type="button"
        aria-current={current ? "step" : undefined}
        onClick={() => open(step)}
        className="group flex w-full items-start gap-3 rounded-md py-2 text-left"
      >
        <WelcomeStepNode state={current ? "current" : mark} />
        <span className="flex min-w-0 flex-col">
          <span
            className={cn(
              "text-sm leading-5 transition-colors duration-150 ease-standard",
              current ? "font-medium text-foreground" : "text-muted-foreground group-hover:text-foreground",
            )}
          >
            {STEP_NAMES[step]}
          </span>
          {!current && mark !== "left" && <span className="font-mono text-[11px] text-muted-foreground/80">{MARK_WORD[mark]}</span>}
        </span>
      </button>
    </li>
  );
};
