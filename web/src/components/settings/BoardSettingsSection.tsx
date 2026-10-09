import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
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
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {statuses && (
        <PageTabs
          label="Board settings"
          tabs={[
            { value: "columns", label: "Status columns" },
            { value: "types", label: "Ticket types" },
            { value: "labels", label: "Label colors" },
          ]}
        >
          <PageTabsContent value="columns">
            <BoardStatusColumnsSection projectId={projectId} statuses={statuses} />
          </PageTabsContent>
          <PageTabsContent value="types">
            <BoardTicketTypesSection projectId={projectId} ticketTypes={ticketTypes} />
          </PageTabsContent>
          <PageTabsContent value="labels">
            <BoardLabelColorsSection projectId={projectId} allLabels={allLabels} labelColors={labelColors} />
          </PageTabsContent>
        </PageTabs>
      )}
    </div>
  );
};
