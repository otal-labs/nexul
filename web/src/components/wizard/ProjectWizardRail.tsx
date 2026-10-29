import { WizardRail } from "@/components/wizard/WizardRail";
import { WizardRailRow } from "@/components/wizard/WizardRailRow";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardProject, useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor, useWizardStepOrder } from "@/hooks/useWizardStepOrder";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { railMark, railRows, wizardDoorWord, wizardStepLabel, type WizardStepId } from "@/models/ProjectWizard";

// "04" for a top-level step, "5.2" for the second sub-step under the fifth.
const railOrdinals = (rows: { nested: boolean }[]): string[] => {
  let top = 0;
  let sub = 0;
  return rows.map((row) => {
    if (!row.nested) {
      top += 1;
      sub = 0;
      return String(top).padStart(2, "0");
    }
    sub += 1;
    return `${top}.${sub}`;
  });
};

interface ProjectWizardRailProps {
  step: WizardStepId;
}

// The door's name over every step as a button, so any step opens at any time; a sub-step sits one indent in.
export const ProjectWizardRail = ({ step }: ProjectWizardRailProps) => {
  const door = useWizardDoor();
  const order = useWizardStepOrder();
  const stack = !!useWizardStackId();
  const setup = useWizardProject()?.setup;
  const goTo = useWizardGoTo();
  const projectName = useProjectWizardStore((s) => s.projectName);
  const arrivedWithProject = useProjectWizardStore((s) => s.projectPreselected);
  const rows = railRows(door, order, step, stack);
  const marks = rows.map((row) => railMark(row.id, { door, order, step, setup, stack }));
  const lastTop = rows.map((row) => row.nested).lastIndexOf(false);
  // Top-level steps only, so the count holds still while Deploy's sub-steps fold in and out; the open step counts by its record.
  const counted = rows.filter((row) => !row.nested);
  const ordinals = railOrdinals(rows);
  const isDone = (id: WizardStepId, index: number) => {
    if (id !== step) return marks[index] === "done";
    return id === "done" || (door === "setup" && railMark(id, { door, order, step: "done", setup, stack }) === "done");
  };

  return (
    <WizardRail
      ariaLabel="Project wizard"
      listLabel="Project wizard steps"
      label={wizardDoorWord(door, arrivedWithProject)}
      name={projectName}
      done={rows.filter((row, index) => !row.nested && isDone(row.id, index)).length}
      total={counted.length}
    >
      {rows.map((row, index) => (
        <WizardRailRow
          key={row.id}
          label={wizardStepLabel(row.id, door)}
          mark={marks[index] ?? "left"}
          ordinal={ordinals[index] ?? ""}
          total={counted.length}
          onOpen={() => goTo(row.id)}
          nested={row.nested}
          nextNested={rows[index + 1]?.nested ?? false}
          spine={row.nested && index < lastTop}
          last={index === rows.length - 1}
        />
      ))}
    </WizardRail>
  );
};
