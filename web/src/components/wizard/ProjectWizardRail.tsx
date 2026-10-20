import { EnterList } from "@/components/EnterList";
import { microheaderClass } from "@/components/Microheader";
import { ProjectWizardRailItem } from "@/components/wizard/ProjectWizardRailItem";
import { useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { railRows, wizardDoorWord, wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface ProjectWizardRailProps {
  step: WizardStepId;
}

// Wide, the door's name and every step as a button; narrow, the door's name over one line naming the current step.
export const ProjectWizardRail = ({ step }: ProjectWizardRailProps) => {
  const door = useWizardDoor();
  const rows = railRows(door, useWizardStepOrder(), step, !!useWizardStackId());
  const projectName = useProjectWizardStore((s) => s.projectName);
  const arrivedWithProject = useProjectWizardStore((s) => s.projectPreselected);
  const position = rows.findIndex((row) => row.id === step) + 1;

  return (
    <nav aria-label="Project wizard" className="@2xl:border-r @2xl:border-border @2xl:pr-8">
      <p className={microheaderClass}>{wizardDoorWord(door, arrivedWithProject)}</p>
      {projectName && <p className="mt-1 truncate font-mono text-xs text-muted-foreground">{projectName}</p>}
      <p className="mt-3 font-mono text-xs tabular-nums @2xl:hidden">
        <span className="font-bold text-foreground">{wizardStepLabel(step, door)}</span>{" "}
        <span className="text-muted-foreground">
          {position} / {rows.length}
        </span>
      </p>
      <EnterList as="ol" arrival="rise" aria-label="Project wizard steps" className="mt-6 hidden @2xl:block">
        {rows.map((row, index) => (
          <ProjectWizardRailItem key={row.id} row={row} step={step} last={index === rows.length - 1} />
        ))}
      </EnterList>
    </nav>
  );
};
