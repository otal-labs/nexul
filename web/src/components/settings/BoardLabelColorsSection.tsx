import { LabelColorRow } from "@/components/settings/LabelColorRow";

interface BoardLabelColorsSectionProps {
  projectId: string;
  allLabels: string[] | undefined;
  labelColors: Record<string, string> | undefined;
}

// Labels aren't a real entity (ADR 0005) — no create/rename/delete, just color on existing ones.
export const BoardLabelColorsSection = ({ projectId, allLabels, labelColors }: BoardLabelColorsSectionProps) => (
  <div className="border-t pt-6">
    <h3 className="text-sm font-semibold">Label colors</h3>
    {allLabels && allLabels.length === 0 && (
      <p className="mt-3 text-sm text-muted-foreground">No labels yet</p>
    )}
    {allLabels && allLabels.length > 0 && (
      <ul className="mt-3 divide-y divide-border">
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
  </div>
);
