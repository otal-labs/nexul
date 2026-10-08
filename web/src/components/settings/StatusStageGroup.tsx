import { PlusIcon } from "lucide-react";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { NewStatusForm } from "@/components/settings/NewStatusForm";
import { StatusRow } from "@/components/settings/StatusRow";
import { Button } from "@/components/ui/button";
import type { BoardStatus, StatusKind } from "@/models/Status";

interface StatusStageGroupProps {
  projectId: string;
  kind: StatusKind;
  label: string;
  // Each row keeps its index in the project's full order, which is what a move swaps.
  rows: { status: BoardStatus; index: number }[];
  total: number;
  adding: boolean;
  onToggleAdding: () => void;
  onMove: (index: number, direction: -1 | 1) => void;
}

export const StatusStageGroup = ({ projectId, kind, label, rows, total, adding, onToggleAdding, onMove }: StatusStageGroupProps) => (
  <div>
    <div className="flex items-center justify-between gap-2 border-b border-border pb-1.5">
      <Microheader>{label}</Microheader>
      <Button variant="ghost" size="icon" className="size-6" aria-label={`Add ${kind} status`} onClick={onToggleAdding}>
        <PlusIcon className="size-3.5" />
      </Button>
    </div>
    {rows.length === 0 && <EmptyRow className="px-0 py-2">No {kind} statuses — skipped on this board</EmptyRow>}
    {rows.length > 0 && (
      // Named for assistive tech since the header states kind once and rows don't repeat it.
      <EnterList aria-label={`${label} statuses`} className="divide-y divide-border">
        {rows.map(({ status, index }) => (
          <StatusRow key={status.id} status={status} index={index} total={total} onMove={(direction) => onMove(index, direction)} />
        ))}
      </EnterList>
    )}
    {adding && <NewStatusForm projectId={projectId} kind={kind} onDone={onToggleAdding} />}
  </div>
);
