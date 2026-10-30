import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LabelColorRow } from "@/components/settings/LabelColorRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { useFetchAllLabels, useFetchLabelColors } from "@/hooks/TicketHooks";

interface BoardLabelColorsSectionProps {
  projectId: string;
}

// Labels aren't a real entity (ADR 0005) — no create/rename/delete, just color on existing ones.
export const BoardLabelColorsSection = ({ projectId }: BoardLabelColorsSectionProps) => {
  const { data: allLabels, isPending, error } = useFetchAllLabels(projectId);
  const { data: labelColors } = useFetchLabelColors(projectId);
  return (
    <SettingsCard id="labels" title="Colors" description="Labels come from tickets. Pick the color each one shows on this board.">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {allLabels && allLabels.length === 0 && <EmptyRow>No labels yet. Add one to a ticket and it shows up here.</EmptyRow>}
      {allLabels && allLabels.length > 0 && (
        <EnterList className={settingsRowListClass}>
          {allLabels.map((label) => (
            <LabelColorRow key={label} label={label} color={labelColors?.[label] ?? ""} projectId={projectId} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
