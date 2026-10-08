import { EmptyRow } from "@/components/EmptyRow";
import { LabelColorRow } from "@/components/settings/LabelColorRow";
import { SettingsCard } from "@/components/settings/SettingsCard";

interface BoardLabelColorsSectionProps {
  projectId: string;
  allLabels: string[] | undefined;
  labelColors: Record<string, string> | undefined;
}

// Labels aren't a real entity (ADR 0005) — no create/rename/delete, just color on existing ones.
export const BoardLabelColorsSection = ({ projectId, allLabels, labelColors }: BoardLabelColorsSectionProps) => (
  <SettingsCard
    id="label-colors"
    title="Label colors"
    description="Labels come from the tickets themselves; pick the color each one shows on this board."
  >
    {allLabels && allLabels.length === 0 && <EmptyRow>No labels yet</EmptyRow>}
    {allLabels && allLabels.length > 0 && (
      <ul className="divide-y divide-border">
        {allLabels.map((label) => (
          <LabelColorRow
            key={label}
            label={label}
            color={labelColors?.[label] ?? ""}
            projectId={projectId}
          />
        ))}
      </ul>
    )}
  </SettingsCard>
);
