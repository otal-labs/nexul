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

  const moveStatus = (index: number, direction: -1 | 1) => {
    const ids = (statuses ?? []).map((s) => s.id);
    const target = index + direction;
    const from = ids[index];
    const to = ids[target];
    if (from === undefined || to === undefined) return;
    ids[index] = to;
    ids[target] = from;
    void reorderStatuses.mutateAsync({ project_id: projectId, ids });
  };

  // Each group's own "+" presets kind — no separate selector; clicking again closes it.
  const toggleAdding = (kind: StatusKind) => setAddingKind(addingKind === kind ? null : kind);

  return (
    <SettingsCard
      id="statuses"
      title="Columns"
      description="Every swimlane on this board shows these columns, grouped under five fixed stages. Leave a stage empty to skip it."
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
              rows={statuses.map((status, index) => ({ status, index })).filter(({ status }) => status.kind === kind)}
              total={statuses.length}
              adding={addingKind === kind}
              onToggleAdding={() => toggleAdding(kind)}
              onMove={moveStatus}
            />
          ))}
        </div>
      )}
    </SettingsCard>
  );
};
