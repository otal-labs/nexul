import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { BoardLabelColorsSection } from "@/components/settings/BoardLabelColorsSection";
import { BoardStatusColumnsSection } from "@/components/settings/BoardStatusColumnsSection";
import { BoardTicketTypesSection } from "@/components/settings/BoardTicketTypesSection";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchAllLabels, useFetchLabelColors } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";

interface BoardSettingsSectionProps {
  projectId: string;
}

// Status columns are project-level: changes here affect every swimlane on the board.
export const BoardSettingsSection = ({ projectId }: BoardSettingsSectionProps) => {
  const { data: statuses, isPending, error } = useFetchProjectStatuses(projectId);
  const { data: ticketTypes } = useFetchProjectTicketTypes(projectId);
  const { data: allLabels } = useFetchAllLabels();
  const { data: labelColors } = useFetchLabelColors(projectId);

  return (
    <div>
      <div className="border-b border-border pb-4">
        <h2 className="text-lg font-semibold tracking-tight">Board columns & types</h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          Columns sit under five fixed stages (Backlog, Progress, Review, Testing, Done) and are
          project-level: every swimlane on this project&apos;s board shows the same columns. Leave a stage
          empty to skip it on this project. A column still holding tickets cannot be removed.
        </p>
      </div>

      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {statuses && (
        <div className="mt-6 space-y-6">
          <BoardStatusColumnsSection projectId={projectId} statuses={statuses} />
          <BoardTicketTypesSection projectId={projectId} ticketTypes={ticketTypes} />
          <BoardLabelColorsSection projectId={projectId} allLabels={allLabels} labelColors={labelColors} />
        </div>
      )}
    </div>
  );
};
