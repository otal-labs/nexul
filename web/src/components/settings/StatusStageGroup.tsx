import { closestCenter, DndContext, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { NewStatusForm } from "@/components/settings/NewStatusForm";
import { StatusRow } from "@/components/settings/StatusRow";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useRowSortSensors } from "@/hooks/useRowSortSensors";
import type { BoardStatus, StatusKind } from "@/models/Status";

interface StatusStageGroupProps {
  projectId: string;
  kind: StatusKind;
  label: string;
  statuses: BoardStatus[];
  adding: boolean;
  onToggleAdding: () => void;
  // Moves one status to where another sits, both in this stage.
  onReorder: (activeId: string, overId: string) => void;
}

export const StatusStageGroup = ({ projectId, kind, label, statuses, adding, onToggleAdding, onReorder }: StatusStageGroupProps) => {
  const sensors = useRowSortSensors();
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (over && active.id !== over.id) onReorder(String(active.id), String(over.id));
  };
  const neighbour = (index: number) => statuses[index]?.id;
  return (
    <div>
      <div className="flex h-8 items-center justify-between gap-2">
        <Microheader>{label}</Microheader>
        <ActionButton permission="projects:write" projectId={projectId} verb="add a status" variant="ghost" size="sm" icon={PlusIcon} className="w-8 px-0" aria-label={`Add ${kind} status`} onClick={onToggleAdding} />
      </div>
      {statuses.length === 0 && (
        <EmptyRow className="-mx-2 flex min-h-11 items-center border-t border-border px-2 py-0">No {kind} statuses, so the board skips this stage.</EmptyRow>
      )}
      {statuses.length > 0 && (
        // Named for assistive tech since the header states kind once and rows don't repeat it.
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
          <SortableContext items={statuses.map((s) => s.id)} strategy={verticalListSortingStrategy}>
            <EnterList aria-label={`${label} statuses`} className={settingsRowListClass}>
              {statuses.map((status, index) => (
                <StatusRow
                  key={status.id}
                  projectId={projectId}
                  status={status}
                  first={index === 0}
                  last={index === statuses.length - 1}
                  onMoveUp={() => onReorder(status.id, neighbour(index - 1) ?? status.id)}
                  onMoveDown={() => onReorder(status.id, neighbour(index + 1) ?? status.id)}
                />
              ))}
            </EnterList>
          </SortableContext>
        </DndContext>
      )}
      {adding && <NewStatusForm projectId={projectId} kind={kind} onDone={onToggleAdding} />}
    </div>
  );
};
