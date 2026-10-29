import { WizardRail } from "@/components/wizard/WizardRail";
import { WizardRailRow } from "@/components/wizard/WizardRailRow";
import type { WizardRailMark } from "@/models/WizardRail";

export type ImportStage = "source" | "adopt" | "done";

const STAGES: { key: ImportStage; label: string }[] = [
  { key: "source", label: "Source" },
  { key: "adopt", label: "Adopt" },
  { key: "done", label: "Done" },
];

interface ImportRailProps {
  stage: ImportStage;
}

// No row opens a stage: Adopt needs the discovery and Done the import, so the flow only moves through its bar.
export const ImportRail = ({ stage }: ImportRailProps) => {
  const at = STAGES.findIndex((s) => s.key === stage);
  const markOf = (index: number): WizardRailMark => {
    if (index === at) return "current";
    if (index < at || stage === "done") return "done";
    return "left";
  };
  const done = STAGES.filter((_, i) => markOf(i) === "done").length + (stage === "done" ? 1 : 0);
  return (
    <WizardRail ariaLabel="Import steps" label="Import" done={done} total={STAGES.length}>
      {STAGES.map((s, i) => (
        <WizardRailRow
          key={s.key}
          label={s.label}
          mark={markOf(i)}
          ordinal={String(i + 1).padStart(2, "0")}
          total={STAGES.length}
          last={i === STAGES.length - 1}
        />
      ))}
    </WizardRail>
  );
};
