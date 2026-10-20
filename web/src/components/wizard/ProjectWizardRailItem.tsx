import { WelcomeStepNode } from "@/components/welcome/WelcomeStepNode";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardProject, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { railMark, wizardStepLabel, type RailRow, type WizardStepId } from "@/models/ProjectWizard";
import { cn } from "@/lib/utils";

interface ProjectWizardRailItemProps {
  row: RailRow;
  step: WizardStepId;
  last: boolean;
}

// Every step is a button, whatever happened there, so any step opens at any time; a sub-step sits one indent in.
export const ProjectWizardRailItem = ({ row, step, last }: ProjectWizardRailItemProps) => {
  const goTo = useWizardGoTo();
  const door = useWizardDoor();
  const setup = useWizardProject()?.setup;
  const stack = !!useWizardStackId();
  const order = useWizardStepOrder();
  const mark = railMark(row.id, { door, order, step, setup, stack });
  const current = mark === "current";

  return (
    <li className={cn("relative", row.nested && "pl-8")} data-state={mark === "left" ? "unvisited" : mark}>
      {!last && <span aria-hidden className={cn("absolute top-7 bottom-0 w-px bg-border", row.nested ? "left-[calc(2rem+9.5px)]" : "left-[9.5px]")} />}
      <button
        type="button"
        aria-current={current ? "step" : undefined}
        onClick={() => goTo(row.id)}
        className="group flex w-full items-start gap-3 rounded-md py-2 text-left"
      >
        <WelcomeStepNode state={mark === "later" ? "skipped" : mark} />
        <span className="flex min-w-0 flex-col">
          <span
            className={cn(
              "text-sm leading-5 transition-colors duration-150 ease-standard",
              current ? "font-medium text-foreground" : "text-muted-foreground group-hover:text-foreground",
            )}
          >
            {wizardStepLabel(row.id, door)}
          </span>
          {!current && mark !== "left" && <span className="font-mono text-[11px] text-muted-foreground/80">{mark}</span>}
        </span>
      </button>
    </li>
  );
};
