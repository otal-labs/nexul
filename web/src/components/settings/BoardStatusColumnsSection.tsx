import { useState } from "react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { StatusStageGroup } from "@/components/settings/StatusStageGroup";
import { useReorderStatuses } from "@/hooks/StatusHooks";
import { STATUS_STAGES, type BoardStatus, type StatusKind } from "@/models/Status";

interface BoardStatusColumnsSectionProps {
  projectId: string;
  statuses: BoardStatus[];
}

export const BoardStatusColumnsSection = ({ projectId, statuses }: BoardStatusColumnsSectionProps) => {
  const reorderStatuses = useReorderStatuses();
  const [addingKind, setAddingKind] = useState<StatusKind | null>(null);

  const moveStatus = (index: number, direction: -1 | 1) => {
    const ids = statuses.map((s) => s.id);
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
      id="status-columns"
      title="Status columns"
      description="Every swimlane on this board shows these columns, grouped under five fixed stages. Leave a stage empty to skip it. A column with tickets in it can't be removed."
    >
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
    </SettingsCard>
  );
};
