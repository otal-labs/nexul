import { arrayMove } from "@dnd-kit/sortable";
import { useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { StatusStageGroup } from "@/components/settings/StatusStageGroup";
import { useFetchProjectStatuses, useReorderStatuses } from "@/hooks/StatusHooks";
import { STATUS_STAGES, type StatusKind } from "@/models/Status";

interface BoardStatusColumnsSectionProps {
  projectId: string;
}

// Status columns are project-level: changes here affect every swimlane on the board.
export const BoardStatusColumnsSection = ({ projectId }: BoardStatusColumnsSectionProps) => {
  const { data: statuses, isPending, error } = useFetchProjectStatuses(projectId);
  const reorderStatuses = useReorderStatuses();
  const [addingKind, setAddingKind] = useState<StatusKind | null>(null);

  const reorder = (activeId: string, overId: string) => {
    const ids = (statuses ?? []).map((s) => s.id);
    const from = ids.indexOf(activeId);
    const to = ids.indexOf(overId);
    if (from < 0 || to < 0 || from === to) return;
    void reorderStatuses.mutateAsync({ project_id: projectId, ids: arrayMove(ids, from, to) });
  };

  // Each group's own "+" presets kind — no separate selector; clicking again closes it.
  const toggleAdding = (kind: StatusKind) => setAddingKind(addingKind === kind ? null : kind);

  return (
    <SettingsCard
      id="statuses"
      title="Columns"
      description="Every swimlane on this board shows these columns, grouped under five fixed stages. Leave a stage empty to skip it, and drag a row to reorder it within its stage."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {statuses && (
        <div className="space-y-6">
          {STATUS_STAGES.map(({ kind, label }) => (
            <StatusStageGroup
              key={kind}
              projectId={projectId}
              kind={kind}
              label={label}
              statuses={statuses.filter((status) => status.kind === kind)}
              adding={addingKind === kind}
              onToggleAdding={() => toggleAdding(kind)}
              onReorder={reorder}
            />
          ))}
        </div>
      )}
    </SettingsCard>
  );
};
